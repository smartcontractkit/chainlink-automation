package conditional

import (
	"context"
	"fmt"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v4/workflows"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/preprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for sampling conditional flow
	samplingConditionInterval = 3 * time.Second
	// Maximum number of upkeeps to be sampled in every round
	maxSampledConditionals = 300
)

type shuffler[T any] interface {
	Shuffle([]T) []T
}

type conditionalProposalFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	upkeepProvider common.ConditionalUpkeepProvider
	shuffler       shuffler[common.UpkeepPayload]
	ratio          types.Ratio
	logger         *log.Logger
}

func NewConditionalProposalFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	ratio types.Ratio,
	upkeepProvider common.ConditionalUpkeepProvider,
	metadataStore types.MetadataStore,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	preProcessors = append(preProcessors, preprocessors.NewProposalFilterer(metadataStore, types.LogTrigger))
	postProcessors := postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore)

	lggr := log.New(logger.Writer(), fmt.Sprintf("[%s | sample-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags)

	workflowProvider := &conditionalProposalFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		upkeepProvider: upkeepProvider,
		ratio:          ratio,
		shuffler:       random.Shuffler[common.UpkeepPayload]{Source: random.NewCryptoRandSource()},

		logger: lggr,
	}

	return workflows.NewPipeline(workflowProvider, samplingConditionInterval, lggr)
}

func (t *conditionalProposalFlow) GetPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
	upkeeps, err := t.upkeepProvider.GetActiveUpkeeps(ctx)
	if err != nil {
		return nil, err
	}

	if len(upkeeps) == 0 {
		return nil, nil
	}

	upkeeps = t.shuffler.Shuffle(upkeeps)
	size := t.ratio.OfInt(len(upkeeps))

	if size <= 0 {
		return nil, nil
	}
	if size > maxSampledConditionals {
		t.logger.Printf("Required sample size %d exceeds max allowed conditional samples %d, limiting to max", size, maxSampledConditionals)
		size = maxSampledConditionals
	}
	if len(upkeeps) < size {
		size = len(upkeeps)
	}
	t.logger.Printf("sampled %d upkeeps", size)
	return upkeeps[:size], nil
}

func (t *conditionalProposalFlow) GetPreprocessors() []ocr2keepersv3.PreProcessor {
	return t.preprocessors
}

func (t *conditionalProposalFlow) GetPostprocessor() postprocessors.PostProcessor {
	return t.postprocessors
}

func (t *conditionalProposalFlow) GetRunner() ocr2keepersv3.Runner {
	return t.runner
}
