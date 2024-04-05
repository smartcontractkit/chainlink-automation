package log

import (
	"context"
	"fmt"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for log trigger flow
	logCheckInterval = 1 * time.Second
)

type logTriggerFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	logProvider    common.LogEventProvider

	upkeepProvider common.ConditionalUpkeepProvider
	ratio          types.Ratio

	logger *log.Logger
}

func NewLogTriggerFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	logProvider common.LogEventProvider,
	retryQ types.RetryQueue,
	resultStore types.ResultStore,
	stateUpdater common.UpkeepStateUpdater,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	postProcessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "log-trigger-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "log-trigger-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &logTriggerFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		logProvider:    logProvider,
		logger:         lggr,
	}

	return workflows.NewPipeline(workflowProvider, logCheckInterval, lggr)
}

func (t *logTriggerFlow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	logs, err := t.logProvider.GetLatestPayloads(ctx)

	t.logger.Printf("%d logs returned by log provider", len(logs))

	return logs, err
}

func (t *logTriggerFlow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *logTriggerFlow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *logTriggerFlow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
