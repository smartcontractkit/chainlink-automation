package v2

import (
	"context"
	"fmt"
	"github.com/smartcontractkit/chainlink-automation/internal/util"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/preprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for sampling conditional flow
	samplingConditionInterval = 3 * time.Second
	// Maximum number of upkeeps to be sampled in every round
	maxSampledConditionals = 300

	observationProcessLimit = 20 * time.Second
)

type shuffler[T any] interface {
	Shuffle([]T) []T
}

type sampleProposalFlow struct {
	preprocessors  []ocr2keepersv3.PreProcessor
	postprocessors postprocessors.PostProcessor
	runner         ocr2keepersv3.Runner
	closer         util.Closer
	upkeepProvider common.ConditionalUpkeepProvider
	shuffler       shuffler[common.UpkeepPayload]
	ratio          types.Ratio

	logger *log.Logger
}

func NewSampleProposalFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	ratio types.Ratio,
	upkeepProvider common.ConditionalUpkeepProvider,
	metadataStore types.MetadataStore,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	preProcessors = append(preProcessors, preprocessors.NewProposalFilterer(metadataStore, types.LogTrigger))
	postProcessors := postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore)

	return &sampleProposalFlow{
		preprocessors:  preProcessors,
		postprocessors: postProcessors,
		runner:         runner,
		upkeepProvider: upkeepProvider,
		ratio:          ratio,
		shuffler:       random.Shuffler[common.UpkeepPayload]{Source: random.NewCryptoRandSource()},

		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | sample-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

func (t *sampleProposalFlow) Start(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	if !t.closer.Store(cancel) {
		return fmt.Errorf("already running")
	}

	t.logger.Printf("starting ticker service")
	defer t.logger.Printf("ticker service stopped")

	ticker := time.NewTicker(samplingConditionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			payloads, err := t.getPayloads(ctx)
			if err != nil {
				t.logger.Printf("error fetching tick: %s", err.Error())
			}
			// observer.Process can be a heavy call taking upto ObservationProcessLimit seconds
			// so it is run in a separate goroutine to not block further ticks
			// Exploratory: Add some control to limit the number of goroutines spawned
			go func(ctx context.Context, payloads []common.UpkeepPayload, l *log.Logger) {
				if err := t.Process(ctx, payloads); err != nil {
					l.Printf("error processing observer: %s", err.Error())
				}
			}(ctx, payloads, t.logger)
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *sampleProposalFlow) Process(ctx context.Context, payloads []common.UpkeepPayload) error {
	pCtx, cancel := context.WithTimeout(ctx, observationProcessLimit)
	defer cancel()

	t.logger.Printf("got %d payloads from ticker", len(payloads))

	var err error

	// Run pre-processors
	for _, preprocessor := range t.preprocessors {
		payloads, err = preprocessor.PreProcess(pCtx, payloads)
		if err != nil {
			return err
		}
	}

	t.logger.Printf("processing %d payloads", len(payloads))

	// Run check pipeline
	results, err := t.runner.CheckUpkeeps(pCtx, payloads...)
	if err != nil {
		return err
	}

	t.logger.Printf("post-processing %d results", len(results))

	// Run post-processor
	if err := t.postprocessors.PostProcess(pCtx, results, payloads); err != nil {
		return err
	}

	t.logger.Printf("finished processing of %d results: %+v", len(results), results)

	return nil
}

func (t *sampleProposalFlow) getPayloads(ctx context.Context) ([]common.UpkeepPayload, error) {
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

func (t *sampleProposalFlow) Close() error {
	_ = t.closer.Close()
	return nil
}
