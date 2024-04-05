package recovery

import (
	"context"
	"fmt"
	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"log"
	"time"
)

const (
	// This is the ticker interval for recovery final flow
	recoveryFinalInterval = 1 * time.Second
	// These are the maximum number of log upkeeps dequeued on every tick from proposal queue in FinalRecoveryFlow
	// This is kept same as OutcomeSurfacedProposalsLimit as those many can get enqueued by plugin in every round
	finalRecoveryBatchSize = 50
)

type finalFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	retryQ         types.RetryQueue
	proposalQ      types.ProposalQueue
	builder        common.PayloadBuilder
	logger         *log.Logger
}

func NewFinalRecoveryFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	resultStore types.ResultStore,
	stateUpdater common.UpkeepStateUpdater,
	retryQ types.RetryQueue,
	proposalQ types.ProposalQueue,
	builder common.PayloadBuilder,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	postProcessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "recovery-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "recovery-final-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &finalFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		retryQ:         retryQ,
		proposalQ:      proposalQ,
		builder:        builder,
		logger:         lggr,
	}

	return workflows.NewPipeline(workflowProvider, recoveryFinalInterval, lggr)
}

func (t *finalFlow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	proposals, err := t.proposalQ.Dequeue(types.LogTrigger, finalRecoveryBatchSize)
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

func (t *finalFlow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *finalFlow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *finalFlow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
