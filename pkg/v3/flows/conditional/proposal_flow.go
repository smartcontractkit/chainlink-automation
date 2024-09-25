package conditional

import (
	"context"
	"fmt"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/preprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
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

type sampler struct {
	logger *log.Logger

	ratio    types.Ratio
	getter   common.ConditionalUpkeepProvider
	shuffler shuffler[common.UpkeepPayload]
}

func NewSampleProposalFlow(
	pre []ocr2keepersv3.PreProcessor,
	ratio types.Ratio,
	getter common.ConditionalUpkeepProvider,
	metadataStore types.MetadataStore,
	runner ocr2keepersv3.Runner,
	logger *log.Logger,
) service.Recoverable {
	pre = append(pre, preprocessors.NewProposalFilterer(metadataStore, types.LogTrigger))
	postprocessors := postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore)

	observer := ocr2keepersv3.NewRunnableObserver(
		pre,
		postprocessors,
		runner,
		observationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | sample-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	getterFn := func(ctx context.Context, _ time.Time) (tickers.Tick, error) {
		return &sampler{
			logger:   logger,
			getter:   getter,
			ratio:    ratio,
			shuffler: random.Shuffler[common.UpkeepPayload]{Source: random.NewCryptoRandSource()},
		}, nil
	}

	lggrPrefix := fmt.Sprintf("[%s | sample-proposal-ticker]", telemetry.ServiceName)
	lggr := log.New(logger.Writer(), lggrPrefix, telemetry.LogPkgStdFlags)

	return tickers.NewTimeTicker(samplingConditionInterval, observer, getterFn, lggr)
}

func (s *sampler) Value(ctx context.Context) ([]common.UpkeepPayload, error) {
	upkeeps, err := s.getter.GetActiveUpkeeps(ctx)
	if err != nil {
		return nil, err
	}
	if len(upkeeps) == 0 {
		return nil, nil
	}

	upkeeps = s.shuffler.Shuffle(upkeeps)
	size := s.ratio.OfInt(len(upkeeps))

	if size <= 0 {
		return nil, nil
	}
	if size > maxSampledConditionals {
		s.logger.Printf("Required sample size %d exceeds max allowed conditional samples %d, limiting to max", size, maxSampledConditionals)
		size = maxSampledConditionals
	}
	if len(upkeeps) < size {
		size = len(upkeeps)
	}
	s.logger.Printf("sampled %d upkeeps", size)
	return upkeeps[:size], nil
}
