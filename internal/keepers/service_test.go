package keepers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestOnDemandUpkeepService(t *testing.T) {
	t.Run("SampleUpkeeps_RatioPick", func(t *testing.T) {
		ctx := context.Background()
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{})
		headsCh := make(chan<- *ethtypes.Header, 1)
		header := &ethtypes.Header{
			Number: big.NewInt(1),
		}

		hs.On("SubscribeNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				switch ch := args.Get(1).(type) {
				case chan<- *ethtypes.Header:
					headsCh = ch
				default:
					t.Error("unexpected chan type")
				}
				subscribed <- struct{}{}
			}).Return(&rpc.ClientSubscription{}, nil)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
			rg.Mock.On("IdentifierFromKey", actives[i]).
				Return(ktypes.UpkeepIdentifier(fmt.Sprintf("%d", i+1)), nil).
				Maybe()
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, ktypes.BlockKey(header.Number.String())).
			Return(actives, nil)

		returnResults := make(ktypes.UpkeepResults, 5)
		for i := 0; i < 5; i++ {
			state := types.NotEligible
			pData := []byte{}
			if i%3 == 0 {
				state = types.Eligible
				pData = []byte(fmt.Sprintf("%d", i))
			}
			returnResults[i] = ktypes.UpkeepResult{
				Key:         actives[i],
				State:       state,
				PerformData: pData,
			}

		}
		rg.Mock.On("CheckUpkeep", mock.Anything, actives[0], actives[1], actives[2], actives[3], actives[4]).
			Return(returnResults, nil)

		l := log.New(io.Discard, "", 0)
		svc := &onDemandUpkeepService{
			logger:         l,
			headSubscriber: hs,
			ratio:          sampleRatio(0.5),
			registry:       rg,
			shuffler:       new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:          newCache[ktypes.UpkeepResult](1 * time.Second),
			cacheCleaner: &intervalCacheCleaner[types.UpkeepResult]{
				Interval: time.Second,
				stop:     make(chan struct{}),
			},
			workers: newWorkerGroup[ktypes.UpkeepResults](2, 10),
		}

		// Start all required processes
		svc.start()

		// Sent head to the head subscriber so
		// the service could sample upkeeps for the given head
		<-subscribed
		headsCh <- header

		// Wait until eligible keys are populated
		for i := 0; i < 5; i++ {
			if len(svc.eligibleUpkeepKeys) > 0 {
				break
			}
			time.Sleep(time.Millisecond * 500)
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

		rg.AssertExpectations(t)
		hs.AssertExpectations(t)
	})

	t.Run("SampleUpkeeps_GetKeysError", func(t *testing.T) {
		ctx := context.Background()
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{})
		headsCh := make(chan<- *ethtypes.Header, 1)
		header := &ethtypes.Header{
			Number: big.NewInt(1),
		}

		hs.On("SubscribeNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				switch ch := args.Get(1).(type) {
				case chan<- *ethtypes.Header:
					headsCh = ch
				default:
					t.Error("unexpected chan type")
				}
				subscribed <- struct{}{}
			}).Return(&rpc.ClientSubscription{}, nil)

		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey(header.Number.String())).
			Run(func(args mock.Arguments) {
				subscribed <- struct{}{}
			}).
			Return([]ktypes.UpkeepKey{}, fmt.Errorf("contract error"))

		logWriter := bytes.NewBuffer(nil)
		l := log.New(logWriter, "", 0)
		svc := &onDemandUpkeepService{
			logger:         l,
			registry:       rg,
			headSubscriber: hs,
			cache:          newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			cacheCleaner: &intervalCacheCleaner[types.UpkeepResult]{
				Interval: time.Second,
				stop:     make(chan struct{}),
			},
			workers: newWorkerGroup[ktypes.UpkeepResults](2, 10),
		}

		// Start background processes
		svc.start()

		// Sent head to the head subscriber so
		// the service could sample upkeeps for the given head
		<-subscribed
		headsCh <- header

		// Wait until GetActiveUpkeepKeys has executed
		<-subscribed

		// this test does not include the cache cleaner or log subscriber
		result, err := svc.SampleUpkeeps(ctx)
		require.NoError(t, err)

		assert.Equal(t, "contract error: failed to get upkeeps from registry for sampling\n", logWriter.String())
		assert.Empty(t, result)

		rg.Mock.AssertExpectations(t)
		hs.AssertExpectations(t)
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
		testId := ktypes.UpkeepIdentifier("1")
		testKey := ktypes.UpkeepKey(fmt.Sprintf("1|%s", string(testId)))

		tests := []struct {
			Name           string
			Ctx            func() (context.Context, func())
			ID             ktypes.UpkeepIdentifier
			Key            ktypes.UpkeepKey
			RegResult      ktypes.UpkeepResults
			Err            error
			ExpectedResult ktypes.UpkeepResults
			ExpectedErr    error
		}{
			{
				Name:           "Result: Skip",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				ID:             testId,
				Key:            testKey,
				RegResult:      ktypes.UpkeepResults{{Key: testKey, State: types.NotEligible}},
				ExpectedResult: ktypes.UpkeepResults{{Key: testKey, State: types.NotEligible}},
			},
			{
				Name:           "Timer Context",
				Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
				ID:             testId,
				Key:            testKey,
				RegResult:      ktypes.UpkeepResults{{Key: testKey, State: types.NotEligible}},
				ExpectedResult: ktypes.UpkeepResults{{Key: testKey, State: types.NotEligible}},
			},
			{
				Name:           "Registry Error",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				ID:             testId,
				Key:            testKey,
				RegResult:      nil,
				Err:            fmt.Errorf("contract error"),
				ExpectedResult: nil,
				ExpectedErr:    fmt.Errorf("contract error: service failed to check upkeep from registry"),
			},
			{
				Name:           "Result: Perform",
				Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
				ID:             testId,
				Key:            testKey,
				RegResult:      ktypes.UpkeepResults{{Key: testKey, State: types.Eligible, PerformData: []byte("1")}},
				ExpectedResult: ktypes.UpkeepResults{{Key: testKey, State: types.Eligible, PerformData: []byte("1")}},
			},
		}

		for i, test := range tests {
			ctx, cancel := test.Ctx()

			rg := ktypes.NewMockRegistry(t)
			rg.Mock.On("IdentifierFromKey", mock.Anything).
				Return(test.ID, nil).
				Maybe()
			rg.Mock.On("CheckUpkeep", mock.Anything, test.Key).
				Return(test.RegResult, test.Err)

			l := log.New(io.Discard, "", 0)
			svc := &onDemandUpkeepService{
				logger:   l,
				cache:    newCache[ktypes.UpkeepResult](20 * time.Millisecond),
				registry: rg,
				workers:  newWorkerGroup[ktypes.UpkeepResults](2, 10),
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
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{})
		headsCh := make(chan<- *ethtypes.Header, 1)
		header := &ethtypes.Header{
			Number: big.NewInt(1),
		}

		hs.On("SubscribeNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				switch ch := args.Get(1).(type) {
				case chan<- *ethtypes.Header:
					headsCh = ch
				default:
					t.Error("unexpected chan type")
				}
				subscribed <- struct{}{}
			}).Return(&rpc.ClientSubscription{}, nil)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
			rg.Mock.On("IdentifierFromKey", actives[i]).
				Return(ktypes.UpkeepIdentifier(fmt.Sprintf("%d", i+1)), nil).
				Maybe()
		}

		rg.Mock.On("GetActiveUpkeepKeys", ctx, ktypes.BlockKey(header.Number.String())).
			Return(actives, nil)

		rg.Mock.On("CheckUpkeep", mock.Anything, actives[0], actives[1], actives[2], actives[3], actives[4]).
			Return(nil, fmt.Errorf("simulate RPC error"))

		l := log.New(io.Discard, "", 0)
		svc := &onDemandUpkeepService{
			logger:         l,
			ratio:          sampleRatio(0.5),
			headSubscriber: hs,
			registry:       rg,
			shuffler:       new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:          newCache[ktypes.UpkeepResult](1 * time.Second),
			cacheCleaner: &intervalCacheCleaner[types.UpkeepResult]{
				Interval: time.Second,
				stop:     make(chan struct{}),
			},
			workers: newWorkerGroup[ktypes.UpkeepResults](2, 10),
		}

		// Start background processes
		svc.start()

		// Sent head to the head subscriber so
		// the service could sample upkeeps for the given head
		<-subscribed
		headsCh <- header

		// Wait until eligible keys are populated
		for i := 0; i < 5; i++ {
			if len(svc.eligibleUpkeepKeys) > 0 {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}

		// this test does not include the cache cleaner or log subscriber
		_, err := svc.SampleUpkeeps(ctx)

		// the process should return an error that wraps the last CheckUpkeep
		// error with context about which key was checked and that too many
		// errors were encountered during the parallel worker process
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many errors in parallel worker process: last error ")

		rg.Mock.AssertExpectations(t)
		hs.AssertExpectations(t)
	})
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
