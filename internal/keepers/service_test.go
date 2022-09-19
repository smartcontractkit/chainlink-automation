package keepers

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestSimpleUpkeepService(t *testing.T) {
	t.Run("SampleUpkeeps_RatioPick", func(t *testing.T) {

		ctx := context.Background()

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
		}

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return(actives, nil)

		for i := 0; i < 5; i++ {
			check := false
			state := types.Skip
			pData := []byte{}
			if i%3 == 0 {
				check = true
				state = types.Perform
				pData = []byte(fmt.Sprintf("%d", i))
			}
			rg.Mock.On("CheckUpkeep", mock.Anything, actives[i]).Return(check, ktypes.UpkeepResult{Key: actives[i], State: state, PerformData: pData}, nil)
		}

		l := log.New(io.Discard, "", 0)
		svc := &simpleUpkeepService{
			logger:   l,
			ratio:    sampleRatio(0.5),
			registry: rg,
			shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:    newCache[ktypes.UpkeepResult](1 * time.Second),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		result, err := svc.SampleUpkeeps(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 2, len(result))

		var matches int
		for _, r := range result {
			for _, a := range actives {
				if string(r.Key) == string(a) {
					matches++
				}
			}
		}

		assert.Equal(t, 2, matches)

		rg.Mock.AssertExpectations(t)
	})

	t.Run("SampleUpkeeps_SkipPreviouslyReported", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		actives := make([]ktypes.UpkeepKey, 2)
		for i := 0; i < 2; i++ {
			actives[i] = ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
		}

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return(actives, nil)
		rg.Mock.On("CheckUpkeep", mock.Anything, actives[1]).Return(false, ktypes.UpkeepResult{Key: actives[1], State: types.Skip}, nil)

		l := log.New(io.Discard, "", 0)
		svc := &simpleUpkeepService{
			logger:   l,
			ratio:    sampleRatio(1.0),
			registry: rg,
			shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		svc.cache.Set(string(actives[0]), ktypes.UpkeepResult{Key: actives[0], State: types.Reported}, defaultExpiration)

		result, err := svc.SampleUpkeeps(ctx)
		cancel()

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))

		rg.Mock.AssertExpectations(t)
	})

	t.Run("SampleUpkeeps_GetKeysError", func(t *testing.T) {
		ctx := context.Background()

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return([]ktypes.UpkeepKey{}, fmt.Errorf("contract error"))

		l := log.New(io.Discard, "", 0)
		svc := &simpleUpkeepService{
			logger:   l,
			registry: rg,
			cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
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
		testKey := ktypes.UpkeepKey([]byte("1|1"))

		tests := []struct {
			Name           string
			Ctx            func() (context.Context, func())
			Key            ktypes.UpkeepKey
			Check          bool
			RegResult      ktypes.UpkeepResult
			Err            error
			ExpectedResult ktypes.UpkeepResult
			ExpectedErr    error
		}{
			{
				Name:           "Result: Skip",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{Key: testKey},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.Skip},
			},
			{
				Name:           "Timer Context",
				Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{Key: testKey},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.Skip},
			},
			{
				Name:           "Registry Error",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{},
				Err:            fmt.Errorf("contract error"),
				ExpectedResult: ktypes.UpkeepResult{},
				ExpectedErr:    fmt.Errorf("contract error"),
			},
			{
				Name:           "Result: Perform",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				Key:            testKey,
				Check:          true,
				RegResult:      ktypes.UpkeepResult{Key: testKey, PerformData: []byte("1")},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.Perform, PerformData: []byte("1")},
			},
		}

		for i, test := range tests {
			ctx, cancel := test.Ctx()

			rg := new(MockedRegistry)
			rg.Mock.On("CheckUpkeep", mock.Anything, test.Key).Return(test.Check, test.RegResult, test.Err)

			l := log.New(io.Discard, "", 0)
			svc := &simpleUpkeepService{
				logger:   l,
				cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
				registry: rg,
				workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
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
		key1 := []byte("test-key-1")
		key2 := []byte("test-key-2")

		tests := []struct {
			Key   []byte
			State ktypes.UpkeepState
			Err   error
		}{
			{Key: key1, State: ktypes.UpkeepState(1), Err: nil},
			{Key: key2, State: ktypes.UpkeepState(2), Err: nil},
		}

		rg := new(MockedRegistry)

		l := log.New(io.Discard, "", 0)
		svc := &simpleUpkeepService{
			logger:   l,
			registry: rg,
			cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		for _, test := range tests {
			ctx := context.Background()
			rg.Mock.On("CheckUpkeep", mock.Anything, ktypes.UpkeepKey(test.Key)).Return(true, ktypes.UpkeepResult{Key: ktypes.UpkeepKey(key1)}, nil)
			err := svc.SetUpkeepState(ctx, ktypes.UpkeepKey(test.Key), test.State)

			if test.Err == nil {
				assert.NoError(t, err, "should not return an error")
			} else {
				assert.Error(t, err, "should return an error")
			}

			assert.Contains(t, svc.cache.data, string(test.Key), "internal state should contain key '%s'", test.Key)
			assert.Equal(t, svc.cache.data[string(test.Key)].Item.State, test.State, "internal state at key '%s' should be %d", test.Key, test.State)
		}
	})
}

type MockedRegistry struct {
	mock.Mock
}

func (_m *MockedRegistry) GetActiveUpkeepKeys(ctx context.Context, key ktypes.BlockKey) ([]ktypes.UpkeepKey, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 []ktypes.UpkeepKey
	if rf, ok := ret.Get(0).(func() []ktypes.UpkeepKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ktypes.UpkeepKey)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedRegistry) CheckUpkeep(ctx context.Context, key ktypes.UpkeepKey) (bool, ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, key)

	var r1 ktypes.UpkeepResult
	if rf, ok := ret.Get(1).(func() ktypes.UpkeepResult); ok {
		r1 = rf()
	} else {
		if ret.Get(0) != nil {
			r1 = ret.Get(1).(ktypes.UpkeepResult)
		}
	}

	return ret.Bool(0), r1, ret.Error(2)
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
