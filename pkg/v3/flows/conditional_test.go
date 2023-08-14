package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestConditionalFinalization(t *testing.T) {
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
	interval := 50 * time.Millisecond
	pre := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	svc := newFinalConditionalFlow(pre, rStore, runner, interval, proposalQ, payloadBuilder, retryQ, upkeepStateUpdater, logger)

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

	time.Sleep(interval*time.Duration(times) + interval/2)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	rStore.AssertExpectations(t)

	wg.Wait()
}

func TestSamplingProposal(t *testing.T) {
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

	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := new(mocks.MockRunnable)
	mStore := new(mocks.MockMetadataStore)
	upkeepProvider := new(mocks.MockConditionalUpkeepProvider)
	ratio := new(mocks.MockRatio)
	blockSub := new(mocks.MockBlockSubscriber)
	coord := new(mocks.MockCoordinator)

	ch := make(chan ocr2keepers.BlockHistory)

	blockSub.On("Subscribe", mock.Anything).Return(1, ch, nil)
	blockSub.On("Unsubscribe", mock.Anything).Return(nil)
	ratio.On("OfInt", mock.Anything).Return(0, nil).Times(1)
	ratio.On("OfInt", mock.Anything).Return(1, nil).Times(1)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
	}, nil).Times(1)

	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.CheckResult{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
			Eligible: true,
		},
	}, nil).Times(1)

	mStore.On("ViewProposals", mock.Anything).Return([]ocr2keepers.CoordinatedBlockProposal{
		{
			UpkeepID: upkeepIDs[2],
			WorkID:   workIDs[2],
		},
	}, nil)
	mStore.On("AddProposals", mock.Anything).Return(nil).Times(1) // should add 1 sample proposal

	upkeepProvider.On("GetActiveUpkeeps", mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(1)
	// set the ticker time lower to reduce the test time
	pre := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	svc, err := newSampleProposalFlow(pre, ratio, upkeepProvider, blockSub, mStore, runner, logger)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		defer wg.Done()
		assert.NoError(t, svc.Start(ctx))
	}(svc, context.Background())

	ch <- []ocr2keepers.BlockKey{
		{
			Number: 2,
			Hash:   [32]byte{2},
		},
		{
			Number: 1,
			Hash:   [32]byte{1},
		},
	}

	ch <- []ocr2keepers.BlockKey{
		{
			Number: 3,
			Hash:   [32]byte{3},
		},
		{
			Number: 2,
			Hash:   [32]byte{2},
		},
		{
			Number: 1,
			Hash:   [32]byte{1},
		},
	}

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	wg.Wait()

	mStore.AssertExpectations(t)
	upkeepProvider.AssertExpectations(t)
	coord.AssertExpectations(t)
	runner.AssertExpectations(t)
}
