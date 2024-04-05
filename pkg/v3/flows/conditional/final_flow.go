package conditional

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for final conditional flow
	finalConditionalInterval = 1 * time.Second
	// These are the maximum number of conditional upkeeps dequeued on every tick from proposal queue in FinalConditionalFlow
	// This is kept same as OutcomeSurfacedProposalsLimit as those many can get enqueued by plugin in every round
	finalConditionalBatchSize = 50

	observationProcessLimit = 20 * time.Second
)

func NewFinalConditionalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[common.UpkeepPayload],
	resultStore types.ResultStore,
	runner ocr2keepersv3.Runner,
	proposalQ types.ProposalQueue,
	builder common.PayloadBuilder,
	retryQ types.RetryQueue,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "conditional-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "conditional-final-retryable-postprocessor")),
	)
	// create observer that only pushes results to result stores. everything at
	// this point can be dropped. this process is only responsible for running
	// conditional proposals that originate from network agreements
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		runner,
		observationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	getterFn := func(ctx context.Context, _ time.Time) (tickers.Tick[[]common.UpkeepPayload], error) {
		return coordinatedProposalsTick{
			logger:    logger,
			builder:   builder,
			q:         proposalQ,
			utype:     types.ConditionTrigger,
			batchSize: finalConditionalBatchSize,
		}, nil
	}

	ticker := tickers.NewTimeTicker[[]common.UpkeepPayload](finalConditionalInterval, observer, getterFn, log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return ticker
}

// coordinatedProposalsTick is used to push proposals from the proposal queue to some observer
type coordinatedProposalsTick struct {
	logger    *log.Logger
	builder   common.PayloadBuilder
	q         types.ProposalQueue
	utype     types.UpkeepType
	batchSize int
}

func (t coordinatedProposalsTick) Value(ctx context.Context) ([]common.UpkeepPayload, error) {
	if t.q == nil {
		return nil, nil
	}

	proposals, err := t.q.Dequeue(t.utype, t.batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue from retry queue: %w", err)
	}
	t.logger.Printf("%d proposals returned from queue", len(proposals))

	builtPayloads, err := t.builder.BuildPayloads(ctx, proposals...)
	if err != nil {
		return nil, fmt.Errorf("failed to build payloads from proposals: %w", err)
	}
	payloads := []common.UpkeepPayload{}
	filtered := 0
	for _, p := range builtPayloads {
		if p.IsEmpty() {
			filtered++
			continue
		}
		payloads = append(payloads, p)
	}
	t.logger.Printf("%d payloads built from %d proposals, %d filtered", len(payloads), len(proposals), filtered)
	return payloads, nil
}
