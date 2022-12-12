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

func TestOnDemandUpkeepService(t *testing.T) {
	t.Run("SampleUpkeeps_RatioPick", func(t *testing.T) {

		ctx := context.Background()
		rg := new(MockedRegistry)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
			rg.Mock.On("IdentifierFromKey", actives[i]).Return(ktypes.UpkeepIdentifier([]byte(fmt.Sprintf("%d", i+1))), nil).Maybe()
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, ktypes.BlockKey("0")).Return(actives, nil)

		for i := 0; i < 5; i++ {
			check := false
			state := types.NotEligible
			pData := []byte{}
			if i%3 == 0 {
				check = true
				state = types.Eligible
				pData = []byte(fmt.Sprintf("%d", i))
			}
			rg.Mock.On("CheckUpkeep", mock.Anything, actives[i]).Return(check, ktypes.UpkeepResult{Key: actives[i], State: state, PerformData: pData}, nil)
		}

		l := log.New(io.Discard, "", 0)
		svc := &onDemandUpkeepService{
			logger:   l,
			ratio:    sampleRatio(0.5),
			registry: rg,
			shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:    newCache[ktypes.UpkeepResult](1 * time.Second),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		// this test does not include the cache cleaner or log subscriber
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

	t.Run("SampleUpkeeps_GetKeysError", func(t *testing.T) {
		ctx := context.Background()

		rg := new(MockedRegistry)
		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return([]ktypes.UpkeepKey{}, fmt.Errorf("contract error"))

		l := log.New(io.Discard, "", 0)
		svc := &onDemandUpkeepService{
			logger:   l,
			registry: rg,
			cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		// this test does not include the cache cleaner or log subscriber
		result, err := svc.SampleUpkeeps(ctx)
		if err == nil {
			t.FailNow()
		}

		assert.Equal(t, fmt.Errorf("contract error: failed to get upkeeps from registry for sampling").Error(), err.Error())
		assert.Equal(t, 0, len(result))

		rg.Mock.AssertExpectations(t)
	})

	/*
		TODO: this test relies on logs and causes race conditions; fix it
		t.Run("SampleUpkeeps_CancelContext", func(t *testing.T) {

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			rg := new(MockedRegistry)

			// test with 100 keys which should start up 100 worker jobs
			keyCount := 100
			actives := make([]ktypes.UpkeepKey, keyCount)
			for i := 0; i < keyCount; i++ {
				actives[i] = ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
				rg.Mock.On("IdentifierFromKey", actives[i]).Return(ktypes.UpkeepIdentifier([]byte(fmt.Sprintf("%d", i+1))), nil).Maybe()
			}

			// the after func simulates a blocking RPC call that either takes
			// 90 milliseconds to complete or exits on context cancellation
			makeAfterFunc := func(ct context.Context) func() chan struct{} {
				return func() chan struct{} {
					c := make(chan struct{}, 1)
					t := time.NewTimer(15 * time.Millisecond)

					go func() {
						select {
						case <-ct.Done():
							t.Stop()
							c <- struct{}{}
						case <-t.C:
							c <- struct{}{}
						}
					}()

					return c
				}
			}

			rg.WithAfter = true
			rg.After = makeAfterFunc(ctx)
			rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return(actives, nil)

			for i := 0; i < keyCount; i++ {
				rg.Mock.On("CheckUpkeep", mock.Anything, actives[i]).
					Return(true, ktypes.UpkeepResult{Key: actives[i], State: types.Eligible, PerformData: []byte{}}, nil).
					Maybe()
			}

			//var logBuff bytes.Buffer
			pR, pW := io.Pipe()

			l := log.New(bufio.NewWriterSize(pW, 100_000_000), "", 0)
			svc := &onDemandUpkeepService{
				logger:   l,
				ratio:    sampleRatio(1.0),
				registry: rg,
				shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
				cache:    newCache[ktypes.UpkeepResult](200 * time.Millisecond),
				workers:  newWorkerGroup[ktypes.UpkeepResult](10, 20),
			}

			result, err := svc.SampleUpkeeps(ctx)
			if err != nil {
				t.FailNow()
			}

			assert.Greater(t, len(result), 0)

			cancel()
			<-time.After(100 * time.Millisecond)

			scnr := bufio.NewScanner(pR)
			scnr.Split(bufio.ScanLines)

			var attempted int
			var notAttempted int
			var cancelled int
			var completed int
			var notSentToWorker int
			for scnr.Scan() {
				line := scnr.Text()

				if strings.Contains(line, "attempting to send") {
					attempted++
				}

				//if strings.Contains(line, "upkeep ready to perform") {
				if strings.Contains(line, "check upkeep took") {
					completed++
				}

				if strings.Contains(line, "check upkeep job context cancelled") {
					cancelled++
				}

				if strings.Contains(line, "job not attempted") {
					notAttempted++
				}

				if strings.Contains(line, "context cancelled while") {
					notSentToWorker++
				}
			}

			t.Logf("jobs attempted: %d", attempted)
			t.Logf("jobs not attempted: %d", notAttempted)
			t.Logf("jobs completed: %d", completed)
			t.Logf("jobs cancelled: %d", cancelled)
			t.Logf("jobs not sent to worker: %d", notSentToWorker)

			assert.Equal(t, attempted, completed+notAttempted+notSentToWorker)
			assert.Greater(t, cancelled, 0)

		})
	*/

	t.Run("CheckUpkeep", func(t *testing.T) {
		testId := ktypes.UpkeepIdentifier([]byte("1"))
		testKey := ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%s", string(testId))))

		tests := []struct {
			Name           string
			Ctx            func() (context.Context, func())
			ID             ktypes.UpkeepIdentifier
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
				ID:             testId,
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{Key: testKey, State: types.NotEligible},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.NotEligible},
			},
			{
				Name:           "Timer Context",
				Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
				ID:             testId,
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{Key: testKey, State: types.NotEligible},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.NotEligible},
			},
			{
				Name:           "Registry Error",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				ID:             testId,
				Key:            testKey,
				Check:          false,
				RegResult:      ktypes.UpkeepResult{},
				Err:            fmt.Errorf("contract error"),
				ExpectedResult: ktypes.UpkeepResult{},
				ExpectedErr:    fmt.Errorf("contract error: service failed to check upkeep from registry"),
			},
			{
				Name:           "Result: Perform",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				ID:             testId,
				Key:            testKey,
				Check:          true,
				RegResult:      ktypes.UpkeepResult{Key: testKey, State: types.Eligible, PerformData: []byte("1")},
				ExpectedResult: ktypes.UpkeepResult{Key: testKey, State: types.Eligible, PerformData: []byte("1")},
			},
		}

		for i, test := range tests {
			ctx, cancel := test.Ctx()

			rg := new(MockedRegistry)
			rg.Mock.On("IdentifierFromKey", mock.Anything).Return(test.ID, nil).Maybe()
			rg.Mock.On("CheckUpkeep", mock.Anything, test.Key).Return(test.Check, test.RegResult, test.Err)

			l := log.New(io.Discard, "", 0)
			svc := &onDemandUpkeepService{
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

	t.Run("SampleUpkeeps_AllChecksReturnError", func(t *testing.T) {
		ctx := context.Background()
		rg := new(MockedRegistry)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1)))
			rg.Mock.On("IdentifierFromKey", actives[i]).Return(ktypes.UpkeepIdentifier([]byte(fmt.Sprintf("%d", i+1))), nil).Maybe()
		}

		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey("0")).Return(actives, nil)

		for i := 0; i < 5; i++ {
			rg.Mock.On("CheckUpkeep", mock.Anything, actives[i]).Return(false, nil, fmt.Errorf("simulate RPC error"))
		}

		l := log.New(io.Discard, "", 0)
		svc := &onDemandUpkeepService{
			logger:   l,
			ratio:    sampleRatio(0.5),
			registry: rg,
			shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:    newCache[ktypes.UpkeepResult](1 * time.Second),
			workers:  newWorkerGroup[ktypes.UpkeepResult](2, 10),
		}

		// this test does not include the cache cleaner or log subscriber
		_, err := svc.SampleUpkeeps(ctx)

		// the process should return an error that wraps the last CheckUpkeep
		// error with context about which key was checked and that too many
		// errors were encountered during the parallel worker process
		assert.Contains(t, err.Error(), "too many errors in parallel worker process: last error ")
	})
}

type MockedRegistry struct {
	mock.Mock
	Ctx       context.Context
	WithAfter bool
	After     func() chan struct{}
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

func (_m *MockedRegistry) CheckUpkeep(ctx context.Context, key ktypes.UpkeepKey, logger *log.Logger) (bool, ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, key)

	var r1 ktypes.UpkeepResult
	if _m.WithAfter {
		select {
		case <-ctx.Done():
			return false, r1, ctx.Err()
		case <-_m.After():
		}
	}

	if ctx.Err() != nil {
		return false, r1, ctx.Err()
	}

	if rf, ok := ret.Get(1).(func() ktypes.UpkeepResult); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(ktypes.UpkeepResult)
		}
	}

	return ret.Bool(0), r1, ret.Error(2)
}

func (_m *MockedRegistry) IdentifierFromKey(key ktypes.UpkeepKey) (ktypes.UpkeepIdentifier, error) {
	ret := _m.Mock.Called(key)

	var r0 ktypes.UpkeepIdentifier
	if rf, ok := ret.Get(0).(func() ktypes.UpkeepIdentifier); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ktypes.UpkeepIdentifier)
		}
	}

	return r0, ret.Error(1)
}

type MockedPerformLogProvider struct {
	mock.Mock
}

func (_m *MockedPerformLogProvider) PerformLogs(ctx context.Context) ([]ktypes.PerformLog, error) {
	ret := _m.Mock.Called(ctx)

	var r0 []ktypes.PerformLog
	if rf, ok := ret.Get(0).(func() []ktypes.PerformLog); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ktypes.PerformLog)
		}
	}

	return r0, ret.Error(1)
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
