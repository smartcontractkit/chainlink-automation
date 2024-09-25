package recovery

import (
	"context"
	"fmt"
	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/preprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"log"
	"time"
)

const (
	// This is the ticker interval for recovery proposal flow
	recoveryProposalInterval = 1 * time.Second
)

type proposalFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	logRecoverer   common.RecoverableProvider
	logger         *log.Logger
}

func NewProposalRecoveryFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	metadataStore types.MetadataStore,
	stateUpdater common.UpkeepStateUpdater,
	runner ocr2keepersv3.Runner,
	logRecoverer common.RecoverableProvider,
	logger *log.Logger,
) service.Recoverable {
	preProcessors = append(preProcessors, preprocessors.NewProposalFilterer(metadataStore, types.LogTrigger))
	postProcessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewIneligiblePostProcessor(stateUpdater, logger),
		postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore),
	)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &proposalFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		logRecoverer:   logRecoverer,
		logger:         lggr,
	}

	return workflows.NewPipeline(workflowProvider, recoveryProposalInterval, lggr)
}

func (t *proposalFlow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	logs, err := t.logRecoverer.GetRecoveryProposals(ctx)

	t.logger.Printf("%d logs returned by log recoverer", len(logs))

	return logs, err
}

func (t *proposalFlow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *proposalFlow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *proposalFlow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
