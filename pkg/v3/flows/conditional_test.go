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

	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/stores"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
)

func TestConditionalFinalization(t *testing.T) {
	upkeepIDs := []common.UpkeepIdentifier{
		common.UpkeepIdentifier([32]byte{1}),
		common.UpkeepIdentifier([32]byte{2}),
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
	proposalQ := stores.NewProposalQueue(func(ui common.UpkeepIdentifier) types.UpkeepType {
		return types.LogTrigger
	})
	upkeepStateUpdater := new(mocks.MockUpkeepStateUpdater)

	retryQ := stores.NewRetryQueue(logger)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]common.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(times)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]common.CheckResult{
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
	payloadBuilder.On("BuildPayloads", mock.Anything, mock.Anything, mock.Anything).Return([]common.UpkeepPayload{
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
	interval := 50 * time.Millisecond
	pre := []ocr2keepersv3.PreProcessor[common.UpkeepPayload]{coord}
	svc := newFinalConditionalFlow(pre, rStore, runner, interval, proposalQ, payloadBuilder, retryQ, upkeepStateUpdater, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	err := proposalQ.Enqueue(common.CoordinatedBlockProposal{
		UpkeepID: upkeepIDs[0],
		WorkID:   workIDs[0],
	}, common.CoordinatedBlockProposal{
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
	upkeepIDs := []common.UpkeepIdentifier{
		common.UpkeepIdentifier([32]byte{1}),
		common.UpkeepIdentifier([32]byte{2}),
		common.UpkeepIdentifier([32]byte{3}),
	}
	workIDs := []string{
		"0x1",
		"0x2",
		"0x3",
	}

	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := mocks.NewMockRunnable(t)
	mStore := mocks.NewMockMetadataStore(t)
	upkeepProvider := mocks.NewMockConditionalUpkeepProvider(t)
	ratio := mocks.NewMockRatio(t)
	coord := mocks.NewMockCoordinator(t)

	ratio.On("OfInt", mock.Anything).Return(0, nil).Times(1)
	ratio.On("OfInt", mock.Anything).Return(1, nil).Times(1)

	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]common.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
	}, nil).Times(1)
	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]common.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(1)
	coord.On("PreProcess", mock.Anything, mock.Anything).Return(nil, nil)

	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]common.CheckResult{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
			Eligible: true,
		},
	}, nil).Times(1)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]common.CheckResult{
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
			Eligible: true,
		},
	}, nil).Times(1)
	runner.On("CheckUpkeeps", mock.Anything).Return(nil, nil)

	mStore.On("ViewProposals", mock.Anything).Return([]common.CoordinatedBlockProposal{
		{
			UpkeepID: upkeepIDs[2],
			WorkID:   workIDs[2],
		},
	}, nil)
	mStore.On("AddProposals", mock.Anything).Return(nil).Times(2)

	upkeepProvider.On("GetActiveUpkeeps", mock.Anything).Return([]common.UpkeepPayload{
		{
			UpkeepID: upkeepIDs[0],
			WorkID:   workIDs[0],
		},
		{
			UpkeepID: upkeepIDs[1],
			WorkID:   workIDs[1],
		},
	}, nil).Times(2)
	upkeepProvider.On("GetActiveUpkeeps", mock.Anything).Return([]common.UpkeepPayload{}, nil)
	// set the ticker time lower to reduce the test time
	pre := []ocr2keepersv3.PreProcessor[common.UpkeepPayload]{coord}
	svc := newSampleProposalFlow(pre, ratio, upkeepProvider, mStore, runner, time.Millisecond*100, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	go func(ctx context.Context) {
		defer wg.Done()
		assert.NoError(t, svc.Start(ctx))
	}(ctx)
	t.Cleanup(func() { assert.NoError(t, svc.Close()) })

	wg.Wait()
}
