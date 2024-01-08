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

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/stores"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
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

	logger := telemetry.NewTelemetryLogger(log.New(io.Discard, "", log.LstdFlags), io.Discard)

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
	upkeepStateUpdater.On("SetUpkeepState", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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

	coord.AssertExpectations(t)
	runner.AssertExpectations(t)
	payloadBuilder.AssertExpectations(t)
	rStore.AssertExpectations(t)

	wg.Wait()
}

func TestRecoveryProposal(t *testing.T) {
	upkeepIDs := []ocr2keepers.UpkeepIdentifier{
		ocr2keepers.UpkeepIdentifier([32]byte{1}),
		ocr2keepers.UpkeepIdentifier([32]byte{2}),
		ocr2keepers.UpkeepIdentifier([32]byte{3}),
	}
	workIDs := []string{
		"0x1",
		"0x2",
		"0x3",
	}

	logger := telemetry.NewTelemetryLogger(log.New(io.Discard, "", log.LstdFlags), io.Discard)

	runner := new(mocks.MockRunnable)
	mStore := new(mocks.MockMetadataStore)
	recoverer := new(mocks.MockRecoverableProvider)
	coord := new(mocks.MockCoordinator)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
	}, nil).Times(1)
	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(1)

	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
			Eligible: true,
		},
	}, nil).Times(1)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
			Eligible: true,
		},
	}, nil).Times(1)

	mStore.On("ViewProposals", mock.Anything).Return([]ocr2keepers.CoordinatedBlockProposal{
		{
			UpkeepID: upkeepIDs[2],
			WorkID:   workIDs[2],
		},
	}, nil).Times(2)
	mStore.On("AddProposals", mock.Anything).Return(nil).Times(2)

	recoverer.On("GetRecoveryProposals", mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
	}, nil).Times(1)
	recoverer.On("GetRecoveryProposals", mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(1)
	// set the ticker time lower to reduce the test time
	interval := 50 * time.Millisecond
	pre := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	stateUpdater := &mockStateUpdater{}
	svc := newRecoveryProposalFlow(pre, runner, mStore, recoverer, interval, stateUpdater, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		defer wg.Done()
		assert.NoError(t, svc.Start(ctx))
	}(svc, context.Background())

	time.Sleep(interval*time.Duration(2) + interval/2)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	mStore.AssertExpectations(t)
	recoverer.AssertExpectations(t)
	runner.AssertExpectations(t)
	coord.AssertExpectations(t)

	wg.Wait()
}

type mockStateUpdater struct {
	ocr2keepers.UpkeepStateUpdater
}
