package conditional

import (
	"context"
	"fmt"
	"github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for final conditional flow
	finalConditionalInterval = 1 * time.Second
	// These are the maximum number of conditional upkeeps dequeued on every tick from proposal queue in FinalConditionalFlow
	// This is kept same as OutcomeSurfacedProposalsLimit as those many can get enqueued by plugin in every round
	finalConditionalBatchSize = 50
)

type conditionalFinalWorkflow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	proposalQ      types.ProposalQueue
	builder        common.PayloadBuilder
	logger         *log.Logger
}

func NewConditionalFinalWorkflow(
	preProcessors []ocr2keepersv3.PreProcessor,
	resultStore types.ResultStore,
	runner ocr2keepersv3.Runner,
	proposalQ types.ProposalQueue,
	builder common.PayloadBuilder,
	retryQ types.RetryQueue,
	logger *log.Logger,
) service.Recoverable {
	postProcessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "conditional-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "conditional-final-retryable-postprocessor")),
	)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-observe]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &conditionalFinalWorkflow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		proposalQ:      proposalQ,
		builder:        builder,
		logger:         lggr,
	}

	return workflows.NewPipeline(workflowProvider, finalConditionalInterval, lggr)
}

func (t *conditionalFinalWorkflow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	proposals, err := t.proposalQ.Dequeue(types.ConditionTrigger, finalConditionalBatchSize)
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

func (t *conditionalFinalWorkflow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *conditionalFinalWorkflow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *conditionalFinalWorkflow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
