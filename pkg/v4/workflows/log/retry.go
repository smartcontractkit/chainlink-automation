package log

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
	// These are the max number of payloads dequeued on every tick from the retry queue in the retry flow
	retryBatchSize = 10

	// This is the ticker interval for retry flow
	retryCheckInterval = 5 * time.Second
)

type retryFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	retryQ         types.RetryQueue
	logger         *log.Logger
}

func NewRetryFlow(
	retryQ types.RetryQueue,
	coord ocr2keepersv3.PreProcessor,
	resultStore types.ResultStore,
	stateUpdater common.UpkeepStateUpdater,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	preProcessors := []ocr2keepersv3.PreProcessor{coord}
	postProcessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "retry-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "retry-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | retry-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &retryFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		logger:         lggr,
		retryQ:         retryQ,
	}

	return workflows.NewPipeline(workflowProvider, retryCheckInterval, lggr)
}

func (t *retryFlow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	payloads, err := t.retryQ.Dequeue(retryBatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue from retry queue: %w", err)
	}
	t.logger.Printf("%d payloads returned by retry queue", len(payloads))

	return payloads, err
}

func (t *retryFlow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *retryFlow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *retryFlow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
