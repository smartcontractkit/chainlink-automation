package keepers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
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
		svcCtx, svcCancel := context.WithCancel(context.Background())
		svc := &onDemandUpkeepService{
			logger:           l,
			cache:            util.NewCache[ktypes.UpkeepResult](20 * time.Millisecond),
			registry:         rg,
			workers:          newWorkerGroup[ktypes.UpkeepResults](2, 10),
			samplingDuration: time.Second * 5,
			ctx:              svcCtx,
			cancel:           svcCancel,
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
	svcCtx, svcCancel := context.WithCancel(context.Background())
	svc := &onDemandUpkeepService{
		logger:           l,
		ratio:            sampleRatio(0.5),
		registry:         rg,
		shuffler:         new(noShuffleShuffler[ktypes.UpkeepKey]),
		cache:            util.NewCache[ktypes.UpkeepResult](1 * time.Second),
		cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](time.Second),
		workers:          newWorkerGroup[ktypes.UpkeepResults](2, 10),
		samplingDuration: time.Second * 5,
		ctx:              svcCtx,
		cancel:           svcCancel,
	}

	svc.samplingResults.set(returnResults)

	// this test does not include the cache cleaner or log subscriber
	result, err := svc.SampleUpkeeps(ctx)
	assert.NoError(t, err)
	assert.Equal(t, returnResults, result)

	rg.AssertExpectations(t)
}

func Test_onDemandUpkeepService_runSamplingUpkeeps(t *testing.T) {
	checkKeys := func(t *testing.T, keys, actualKeys []ktypes.UpkeepKey) {
		for _, key := range keys[:5] {
			var found bool
			for _, actualKey := range actualKeys {
				if bytes.Equal(actualKey, key) {
					found = true
					break
				}
			}
			assert.True(t, found)
		}
	}

	t.Run("successfully sampled upkeeps", func(t *testing.T) {
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{}, 1)
		header := types.BlockKey("1")

		chHeads := make(chan ktypes.BlockKey, 1)
		chHeads <- header
		hs.Mock.On("HeadTicker").Return(chHeads)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, types.BlockKey("0")).
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

		rg.Mock.On("CheckUpkeep", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				checkKeys(t, actives[:5], []ktypes.UpkeepKey{
					args.Get(1).(ktypes.UpkeepKey),
					args.Get(2).(ktypes.UpkeepKey),
					args.Get(3).(ktypes.UpkeepKey),
					args.Get(4).(ktypes.UpkeepKey),
					args.Get(5).(ktypes.UpkeepKey),
				})
				close(subscribed)
			}).
			Return(returnResults, nil)

		l := log.New(io.Discard, "", 0)
		svcCtx, svcCancel := context.WithCancel(context.Background())
		svc := &onDemandUpkeepService{
			logger:           l,
			headSubscriber:   hs,
			ratio:            sampleRatio(0.5),
			registry:         rg,
			shuffler:         new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:            util.NewCache[ktypes.UpkeepResult](1 * time.Second),
			cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](time.Second),
			workers:          newWorkerGroup[ktypes.UpkeepResults](2, 10),
			samplingDuration: time.Second * 5,
			ctx:              svcCtx,
			cancel:           svcCancel,
		}

		// Start all required processes
		svc.start()

		// Wait until upkees are checked
		<-subscribed

		svc.stop()

		// TODO: Use gomega or something similar
		var actualResults types.UpkeepResults
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			actualResults = svc.samplingResults.get()
			if len(actualResults) > 0 {
				break
			}
		}

		assert.Len(t, actualResults, 2)
		assert.Equal(t, returnResults[0], actualResults[0])
		assert.Equal(t, returnResults[3], actualResults[1])

		rg.AssertExpectations(t)
		hs.AssertExpectations(t)
	})

	t.Run("getting active upkeeps error", func(t *testing.T) {
		rg := ktypes.NewMockRegistry(t)
		hs := ktypes.NewMockHeadSubscriber(t)
		subscribed := make(chan struct{}, 1)
		header := types.BlockKey("1")

		chHeads := make(chan ktypes.BlockKey, 1)
		chHeads <- header
		hs.Mock.On("HeadTicker").Return(chHeads)

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, ktypes.BlockKey("0")).
			Run(func(args mock.Arguments) {
				close(subscribed)
			}).
			Return([]ktypes.UpkeepKey{}, fmt.Errorf("contract error"))

		var logWriter buffer
		l := log.New(&logWriter, "", 0)
		svcCtx, svcCancel := context.WithCancel(context.Background())
		svc := &onDemandUpkeepService{
			logger:           l,
			registry:         rg,
			headSubscriber:   hs,
			cache:            util.NewCache[ktypes.UpkeepResult](20 * time.Millisecond),
			cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](time.Second),
			workers:          newWorkerGroup[ktypes.UpkeepResults](2, 10),
			samplingDuration: time.Second * 5,
			ctx:              svcCtx,
			cancel:           svcCancel,
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

		chHeads := make(chan ktypes.BlockKey, 1)
		chHeads <- header
		hs.Mock.On("HeadTicker").Return(chHeads)

		actives := make([]ktypes.UpkeepKey, 10)
		for i := 0; i < 10; i++ {
			actives[i] = ktypes.UpkeepKey(fmt.Sprintf("1|%d", i+1))
		}

		rg.Mock.On("GetActiveUpkeepKeys", mock.Anything, ktypes.BlockKey("0")).
			Return(actives, nil)

		rg.Mock.On("CheckUpkeep", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				checkKeys(t, actives[:5], []ktypes.UpkeepKey{
					args.Get(1).(ktypes.UpkeepKey),
					args.Get(2).(ktypes.UpkeepKey),
					args.Get(3).(ktypes.UpkeepKey),
					args.Get(4).(ktypes.UpkeepKey),
					args.Get(5).(ktypes.UpkeepKey),
				})
				close(subscribed)
			}).
			Return(nil, fmt.Errorf("simulate RPC error"))

		var logWriter buffer
		l := log.New(&logWriter, "", 0)
		svcCtx, svcCancel := context.WithCancel(context.Background())
		svc := &onDemandUpkeepService{
			logger:           l,
			headSubscriber:   hs,
			ratio:            sampleRatio(0.5),
			registry:         rg,
			shuffler:         new(noShuffleShuffler[ktypes.UpkeepKey]),
			cache:            util.NewCache[ktypes.UpkeepResult](1 * time.Second),
			cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](time.Second),
			workers:          newWorkerGroup[ktypes.UpkeepResults](2, 10),
			samplingDuration: time.Second * 5,
			ctx:              svcCtx,
			cancel:           svcCancel,
		}

		// Start background processes
		svc.start()

		// Wait until CheckUpkeep is called
		<-subscribed

		svc.stop()

		// TODO: Get rid of this
		time.Sleep(time.Second)

		assert.Contains(t, logWriter.String(), "simulate RPC error: failed to check upkeep keys:")

		rg.Mock.AssertExpectations(t)
		hs.AssertExpectations(t)
	})
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
