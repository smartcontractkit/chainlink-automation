package keepers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	k_types "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSimpleUpkeepService(t *testing.T) {
	t.Run("SampleUpkeeps_RatioPick", func(t *testing.T) {

		ctx := context.Background()

		actives := make([]types.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = k_types.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
		}

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, k_types.BlockKey("0")).Return(actives, nil)

		for i := 0; i < 5; i++ {
			check := false
			state := Skip
			pData := []byte{}
			if i%3 == 0 {
				check = true
				state = Perform
				pData = []byte(fmt.Sprintf("%d", i))
			}
			rg.Mock.On("CheckUpkeep", ctx, mock.Anything, actives[i]).Return(check, k_types.UpkeepResult{Key: actives[1], State: state, PerformData: pData}, nil)
		}

		svc := &simpleUpkeepService{
			ratio:    SampleRatio(0.5),
			registry: rg,
			shuffler: new(noShuffleShuffler[types.UpkeepKey]),
			state:    make(map[string]types.UpkeepState),
		}

		result, err := svc.SampleUpkeeps(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))

		rg.Mock.AssertExpectations(t)
	})

	t.Run("SampleUpkeeps_SkipPreviouslyReported", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		actives := make([]types.UpkeepKey, 2)
		for i := 0; i < 2; i++ {
			actives[i] = k_types.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
		}

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, k_types.BlockKey("0")).Return(actives, nil)
		rg.Mock.On("CheckUpkeep", ctx, mock.Anything, actives[1]).Return(false, k_types.UpkeepResult{Key: actives[1], State: Skip}, nil)

		svc := &simpleUpkeepService{
			ratio:    SampleRatio(1.0),
			registry: rg,
			shuffler: new(noShuffleShuffler[types.UpkeepKey]),
			state:    make(map[string]types.UpkeepState),
		}

		svc.mu.Lock()
		svc.state[string(actives[0])] = Reported
		svc.mu.Unlock()

		result, err := svc.SampleUpkeeps(ctx)
		cancel()

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))

		rg.Mock.AssertExpectations(t)
	})

	t.Run("SampleUpkeeps_GetKeysError", func(t *testing.T) {
		ctx := context.Background()

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, k_types.BlockKey("0")).Return([]types.UpkeepKey{}, fmt.Errorf("contract error"))

		svc := &simpleUpkeepService{
			registry: rg,
			state:    make(map[string]types.UpkeepState),
		}

		result, err := svc.SampleUpkeeps(ctx)
		if err == nil {
			t.FailNow()
		}

		assert.Equal(t, fmt.Errorf("contract error").Error(), err.Error())
		assert.Equal(t, 0, len(result))

		rg.Mock.AssertExpectations(t)
	})

	t.Run("CheckUpkeep", func(t *testing.T) {
		testKey := k_types.UpkeepKey([]byte("1|1"))

		tests := []struct {
			Name           string
			Ctx            func() (context.Context, func())
			Key            k_types.UpkeepKey
			Check          bool
			RegResult      k_types.UpkeepResult
			Err            error
			ExpectedResult k_types.UpkeepResult
			ExpectedErr    error
		}{
			{
				Name:           "Result: Skip",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          false,
				RegResult:      k_types.UpkeepResult{Key: testKey},
				ExpectedResult: k_types.UpkeepResult{Key: testKey, State: Skip},
			},
			{
				Name:           "Timer Context",
				Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
				Key:            testKey,
				Check:          false,
				RegResult:      k_types.UpkeepResult{Key: testKey},
				ExpectedResult: k_types.UpkeepResult{Key: testKey, State: Skip},
			},
			{
				Name:           "Registry Error",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          false,
				RegResult:      k_types.UpkeepResult{},
				Err:            fmt.Errorf("contract error"),
				ExpectedResult: k_types.UpkeepResult{},
				ExpectedErr:    fmt.Errorf("contract error"),
			},
			{
				Name:           "Result: Perform",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          true,
				RegResult:      k_types.UpkeepResult{Key: testKey, PerformData: []byte("1")},
				ExpectedResult: k_types.UpkeepResult{Key: testKey, State: Perform, PerformData: []byte("1")},
			},
		}

		for i, test := range tests {
			ctx, cancel := test.Ctx()

			rg := new(MockedRegistry)
			rg.Mock.On("CheckUpkeep", ctx, mock.Anything, test.Key).Return(test.Check, test.RegResult, test.Err)

			svc := &simpleUpkeepService{
				state:    make(map[string]types.UpkeepState),
				registry: rg,
			}

			result, err := svc.CheckUpkeep(ctx, test.Key)
			cancel()

			if test.ExpectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, err.Error(), test.ExpectedErr.Error(), "error should match expected for test %d", i+1)
			}

			assert.Equal(t, test.ExpectedResult, result, "result should match expected for test %d", i+1)

			rg.Mock.AssertExpectations(t)
		}
	})

	t.Run("SetUpkeepState", func(t *testing.T) {
		tests := []struct {
			Key   []byte
			State types.UpkeepState
			Err   error
		}{
			{Key: []byte("test-key-1"), State: types.UpkeepState(1), Err: nil},
			{Key: []byte("test-key-2"), State: types.UpkeepState(2), Err: nil},
		}

		svc := &simpleUpkeepService{
			state: make(map[string]types.UpkeepState),
		}

		for _, test := range tests {
			err := svc.SetUpkeepState(context.Background(), types.UpkeepKey(test.Key), test.State)

			if test.Err == nil {
				assert.NoError(t, err, "should not return an error")
			} else {
				assert.Error(t, err, "should return an error")
			}

			assert.Contains(t, svc.state, string(test.Key), "internal state should contain key '%s'", test.Key)
			assert.Equal(t, svc.state[string(test.Key)], test.State, "internal state at key '%s' should be %d", test.Key, test.State)
		}
	})
}

type MockedRegistry struct {
	mock.Mock
}

func (_m *MockedRegistry) GetActiveUpkeepKeys(ctx context.Context, key types.BlockKey) ([]types.UpkeepKey, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 []types.UpkeepKey
	if rf, ok := ret.Get(0).(func() []types.UpkeepKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.UpkeepKey)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedRegistry) CheckUpkeep(ctx context.Context, addr types.Address, key types.UpkeepKey) (bool, types.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, addr, key)

	var r1 types.UpkeepResult
	if rf, ok := ret.Get(1).(func() types.UpkeepResult); ok {
		r1 = rf()
	} else {
		if ret.Get(0) != nil {
			r1 = ret.Get(1).(k_types.UpkeepResult)
		}
	}

	return ret.Bool(0), r1, ret.Error(2)
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
