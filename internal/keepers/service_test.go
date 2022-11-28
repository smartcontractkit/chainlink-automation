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

func Test_onDemandUpkeepService_CheckUpkeep(t *testing.T) {
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
}

func Test_onDemandUpkeepService_SampleUpkeeps(t *testing.T) {
	ctx := context.Background()
	rg := ktypes.NewMockRegistry(t)

	returnResults := make(ktypes.UpkeepResults, 5)
	for i := 0; i < 5; i++ {
		returnResults[i] = ktypes.UpkeepResult{
			Key:         ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1)),
			State:       types.NotEligible,
			PerformData: []byte{},
		}
	}

	l := log.New(io.Discard, "", 0)
	svc := &onDemandUpkeepService{
		logger:   l,
		ratio:    sampleRatio(0.5),
		registry: rg,
		shuffler: new(noShuffleShuffler[ktypes.UpkeepKey]),
		cache:    newCache[ktypes.UpkeepResult](1 * time.Second),
		cacheCleaner: &intervalCacheCleaner[types.UpkeepResult]{
			Interval: time.Second,
			stop:     make(chan struct{}),
		},
		workers: newWorkerGroup[ktypes.UpkeepResults](2, 10),
	}

	svc.samplingResults.set(returnResults)

	// this test does not include the cache cleaner or log subscriber
	result, err := svc.SampleUpkeeps(ctx)
	assert.NoError(t, err)
	assert.Equal(t, returnResults, result)

	rg.AssertExpectations(t)
}

func Test_onDemandUpkeepService_runSamplingUpkeeps(t *testing.T) {
	t.Run("successfully sampled upkeeps", func(t *testing.T) {
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{}, 1)
		header := types.BlockKey("1")
		stopProcs := make(chan struct{})

		hs.On("OnNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				cb, ok := args.Get(1).(func(blockKey types.BlockKey))
				assert.True(t, ok)
				cb(header)
				<-stopProcs
			}).Return(nil)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, header).
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
			Run(func(args mock.Arguments) {
				close(subscribed)
			}).
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
			workers:   newWorkerGroup[ktypes.UpkeepResults](2, 10),
			stopProcs: stopProcs,
		}

		// Start all required processes
		svc.start()

		// Wait until upkees are checked
		<-subscribed

		svc.stop()

		// TODO: Get rid of it
		time.Sleep(time.Second)

		assert.Len(t, svc.samplingResults.get(), 2)
		assert.Equal(t, returnResults[0], svc.samplingResults.get()[0])
		assert.Equal(t, returnResults[3], svc.samplingResults.get()[1])

		rg.AssertExpectations(t)
		hs.AssertExpectations(t)
	})

	t.Run("getting active upkeeps error", func(t *testing.T) {
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{}, 1)
		header := types.BlockKey("1")
		stopProcs := make(chan struct{})

		hs.On("OnNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				cb, ok := args.Get(1).(func(blockKey types.BlockKey))
				assert.True(t, ok)
				cb(header)
				<-stopProcs
			}).Return(nil)

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, header).
			Run(func(args mock.Arguments) {
				subscribed <- struct{}{}
			}).
			Return([]ktypes.UpkeepKey{}, fmt.Errorf("contract error"))

		var logWriter buffer
		l := log.New(&logWriter, "", 0)
		svc := &onDemandUpkeepService{
			logger:         l,
			registry:       rg,
			headSubscriber: hs,
			cache:          newCache[ktypes.UpkeepResult](20 * time.Millisecond),
			cacheCleaner: &intervalCacheCleaner[types.UpkeepResult]{
				Interval: time.Second,
				stop:     make(chan struct{}),
			},
			workers:   newWorkerGroup[ktypes.UpkeepResults](2, 10),
			stopProcs: stopProcs,
		}

		// Start background processes
		svc.start()

		// Wait until GetActiveUpkeepKeys is called
		<-subscribed

		svc.stop()

		assert.Equal(t, "contract error: failed to get upkeeps from registry for sampling\n", logWriter.String())

		rg.Mock.AssertExpectations(t)
		hs.AssertExpectations(t)
	})

	t.Run("getting check upkeeps error", func(t *testing.T) {
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{}, 1)
		header := types.BlockKey("1")
		stopProcs := make(chan struct{})

		hs.On("OnNewHead", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				cb, ok := args.Get(1).(func(blockKey types.BlockKey))
				assert.True(t, ok)
				cb(header)
				<-stopProcs
			}).Return(nil)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, header).
			Return(actives, nil)

		rg.Mock.On("CheckUpkeep", mock.Anything, actives[0], actives[1], actives[2], actives[3], actives[4]).
			Run(func(args mock.Arguments) {
				subscribed <- struct{}{}
			}).
			Return(nil, fmt.Errorf("simulate RPC error"))

		var logWriter buffer
		l := log.New(&logWriter, "", 0)
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
			workers:   newWorkerGroup[ktypes.UpkeepResults](2, 10),
			stopProcs: stopProcs,
		}

		// Start background processes
		svc.start()

		// Wait until CheckUpkeep is called
		<-subscribed

		svc.stop()

		assert.Contains(t, logWriter.String(), "simulate RPC error: failed to check upkeep keys:")

		rg.Mock.AssertExpectations(t)
		hs.AssertExpectations(t)
	})
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
