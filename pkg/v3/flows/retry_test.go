package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/stores"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestRetryFlow(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	times := 3

	runner := new(mocks.MockRunnable)
	rStore := new(mocks.MockResultStore)
	coord := new(mocks.MockCoordinator)
	upkeepStateUpdater := new(mocks.MockUpkeepStateUpdater)
	retryQ := stores.NewRetryQueue(logger)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]common.UpkeepPayload{
		{
			UpkeepID: common.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
		},
		{
			UpkeepID: common.UpkeepIdentifier([32]byte{2}),
			WorkID:   "0x2",
		},
	}, nil).Times(times)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]common.CheckResult{
		{
			UpkeepID: common.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
			Eligible: true,
		},
		{
			UpkeepID:  common.UpkeepIdentifier([32]byte{2}),
			WorkID:    "0x2",
			Retryable: true,
		},
	}, nil).Times(times)
	// within the 3 ticks, it should retry twice and the third time it should be eligible and add to result store
	rStore.On("Add", mock.Anything).Times(times)
	upkeepStateUpdater.On("SetUpkeepState", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// set the ticker time lower to reduce the test time
	retryInterval := 50 * time.Millisecond

	svc := NewRetryFlow(coord, rStore, runner, retryQ, retryInterval, upkeepStateUpdater, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	err := retryQ.Enqueue(types.RetryRecord{
		Payload: common.UpkeepPayload{
			UpkeepID: common.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
		},
	}, types.RetryRecord{
		Payload: common.UpkeepPayload{
			UpkeepID: common.UpkeepIdentifier([32]byte{2}),
			WorkID:   "0x2",
		},
	})
	assert.NoError(t, err)

	go func(svc service.Recoverable, ctx context.Context) {
		assert.NoError(t, svc.Start(ctx))
		wg.Done()
	}(svc, context.Background())

	time.Sleep(retryInterval*time.Duration(times) + retryInterval/2)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	coord.AssertExpectations(t)
	runner.AssertExpectations(t)
	rStore.AssertExpectations(t)

	wg.Wait()
}
