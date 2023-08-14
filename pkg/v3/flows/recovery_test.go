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

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
)

func TestRecoveryFinalization(t *testing.T) {
	upkeepIDs := []ocr2keepers.UpkeepIdentifier{
		ocr2keepers.UpkeepIdentifier([32]byte{1}),
		ocr2keepers.UpkeepIdentifier([32]byte{2}),
	}
	workIDs := []string{
		"0x1",
		"0x2",
	}

	logger := log.New(io.Discard, "", log.LstdFlags)

	times := 3

	runner := new(mocks.MockRunnable)
	rStore := new(mocks.MockResultStore)
	coord := new(mocks.MockCoordinator)
	payloadBuilder := new(mocks.MockPayloadBuilder)
	proposalQ := stores.NewProposalQueue(func(ui ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
		return ocr2keepers.LogTrigger
	})
	upkeepStateUpdater := new(mocks.MockUpkeepStateUpdater)

	retryQ := stores.NewRetryQueue(logger)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(times)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
			Eligible: true,
		},
		{
			UpkeepID:  upkeepIDs[1],
			WorkID:    workIDs[1],
			Retryable: true,
		},
	}, nil).Times(times)
	rStore.On("Add", mock.Anything).Times(times)
	payloadBuilder.On("BuildPayloads", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(times)

	// set the ticker time lower to reduce the test time
	recFinalInterval := 50 * time.Millisecond
	pre := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	svc := newFinalRecoveryFlow(pre, rStore, runner, retryQ, recFinalInterval, proposalQ, payloadBuilder, upkeepStateUpdater, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	err := proposalQ.Enqueue(ocr2keepers.CoordinatedBlockProposal{
		UpkeepID: upkeepIDs[0],
		WorkID:   workIDs[0],
	}, ocr2keepers.CoordinatedBlockProposal{
		UpkeepID: upkeepIDs[1],
		WorkID:   workIDs[1],
	})
	assert.NoError(t, err)

	go func(svc service.Recoverable, ctx context.Context) {
		defer wg.Done()
		assert.NoError(t, svc.Start(ctx))
	}(svc, context.Background())

	time.Sleep(recFinalInterval*time.Duration(times) + recFinalInterval/2)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	rStore.AssertExpectations(t)

	wg.Wait()
}
