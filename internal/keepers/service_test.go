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

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/ratio"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/types/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

func Test_onDemandUpkeepService_CheckUpkeep(t *testing.T) {
	testId := ktypes.UpkeepIdentifier("1")
	testKey := chain.UpkeepKey(fmt.Sprintf("1|%s", string(testId)))

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

		rg := mocks.NewRegistry(t)
		rg.Mock.On("CheckUpkeep", mock.Anything, mock.Anything, test.Key).
			Return(test.RegResult, test.Err)

		l := log.New(io.Discard, "", 0)
		svcCtx, svcCancel := context.WithCancel(context.Background())
		svc := &onDemandUpkeepService{
			logger:           l,
			cache:            util.NewCache[ktypes.UpkeepResult](20 * time.Millisecond),
			registry:         rg,
			workers:          util.NewWorkerGroup[ktypes.UpkeepResults](2, 10),
			samplingDuration: time.Second * 5,
			ctx:              svcCtx,
			cancel:           svcCancel,
		}

		result, err := svc.CheckUpkeep(ctx, false, test.Key)
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
	rg := mocks.NewRegistry(t)

	blockKey := chain.BlockKey("1")
	returnResults := make(ktypes.UpkeepResults, 5)
	for i := 0; i < 5; i++ {
		returnResults[i] = ktypes.UpkeepResult{
			Key:         chain.UpkeepKey(fmt.Sprintf("%s|%d", blockKey, i+1)),
			State:       types.NotEligible,
			PerformData: []byte{},
		}
	}

	l := log.New(io.Discard, "", 0)
	svcCtx, svcCancel := context.WithCancel(context.Background())
	svc := &onDemandUpkeepService{
		logger:           l,
		ratio:            ratio.SampleRatio(0.5),
		registry:         rg,
		shuffler:         new(noShuffleShuffler[ktypes.UpkeepIdentifier]),
		cache:            util.NewCache[ktypes.UpkeepResult](1 * time.Second),
		cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](time.Second),
		workers:          util.NewWorkerGroup[ktypes.UpkeepResults](2, 10),
		samplingDuration: time.Second * 5,
		ctx:              svcCtx,
		cancel:           svcCancel,
	}

	svc.samplingResults.set(blockKey, returnResults)

	// this test does not include the cache cleaner or log subscriber
	bk, result, err := svc.SampleUpkeeps(ctx)
	assert.NoError(t, err)
	assert.Equal(t, returnResults, result)
	assert.Equal(t, blockKey, bk)

	rg.AssertExpectations(t)
}

type noShuffleShuffler[T any] struct{}

func (_ *noShuffleShuffler[T]) Shuffle(a []T) []T {
	return a
}
