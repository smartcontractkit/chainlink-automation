package log

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for recovery final flow
	recoveryFinalInterval = 1 * time.Second
	// These are the maximum number of log upkeeps dequeued on every tick from proposal queue in FinalRecoveryFlow
	// This is kept same as OutcomeSurfacedProposalsLimit as those many can get enqueued by plugin in every round
	finalRecoveryBatchSize = 50
)

func NewFinalRecoveryFlow(
	preprocessors []ocr2keepersv3.PreProcessor,
	resultStore types.ResultStore,
	runner ocr2keepersv3.Runner,
	retryQ types.RetryQueue,
	proposalQ types.ProposalQueue,
	builder common.PayloadBuilder,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "recovery-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "recovery-final-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	observerLggrPrefix := fmt.Sprintf("[%s | recovery-final-observer]", telemetry.ServiceName)
	observerLggr := log.New(logger.Writer(), observerLggrPrefix, telemetry.LogPkgStdFlags)

	// create observer that only pushes results to result stores. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	recoveryObserver := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		runner,
		observationProcessLimit,
		observerLggr,
	)

	getterFn := func(ctx context.Context, _ time.Time) (tickers.Tick, error) {
		return coordinatedProposalsTick{
			logger:    logger,
			builder:   builder,
			q:         proposalQ,
			utype:     types.LogTrigger,
			batchSize: finalRecoveryBatchSize,
		}, nil
	}

	lggrPrefix := fmt.Sprintf("[%s | recovery-final-ticker]", telemetry.ServiceName)
	lggr := log.New(logger.Writer(), lggrPrefix, telemetry.LogPkgStdFlags)

	ticker := tickers.NewTimeTicker(recoveryFinalInterval, recoveryObserver, getterFn, lggr)

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