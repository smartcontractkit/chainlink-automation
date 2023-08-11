package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

//go:generate mockery --name Ratio --structname MockRatio --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename ratio.generated.go
type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
}

// ConditionalEligibility is a flow controller that surfaces conditional upkeeps
type ConditionalEligibility struct {
	builder ocr2keepers.PayloadBuilder
	mStore  ocr2keepers.MetadataStore
	logger  *log.Logger
}

// NewConditionalEligibility ...
func NewConditionalEligibility(
	ratio Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	subscriber ocr2keepers.BlockSubscriber,
	builder ocr2keepers.PayloadBuilder,
	rs ResultStore,
	ms ocr2keepers.MetadataStore,
	rn ocr2keepersv3.Runner,
	proposalQ ocr2keepers.ProposalQueue,
	retryQ ocr2keepers.RetryQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	typeGetter ocr2keepers.UpkeepTypeGetter,
	logger *log.Logger,
) (*ConditionalEligibility, []service.Recoverable, error) {
	// TODO: add coordinator to preprocessor list
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{}

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	svc0 := newFinalConditionalFlow(preprocessors, rs, rn, time.Second, proposalQ, builder, retryQ, stateUpdater, logger)

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	svc1, err := newSampleProposalFlow(preprocessors, ratio, getter, subscriber, ms, rn, typeGetter, logger)
	if err != nil {
		return nil, nil, err
	}

	return &ConditionalEligibility{
		mStore:  ms,
		builder: builder,
		logger:  logger,
	}, []service.Recoverable{svc0, svc1}, err
}

func newSampleProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ratio Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	subscriber ocr2keepers.BlockSubscriber,
	ms ocr2keepers.MetadataStore,
	runner ocr2keepersv3.Runner,
	typeGetter ocr2keepers.UpkeepTypeGetter,
	logger *log.Logger,
) (service.Recoverable, error) {
	preprocessors = append(preprocessors, &proposalFilterer{ms, ocr2keepers.LogTrigger})
	postprocessors := postprocessors.NewAddProposalToMetadataStorePostprocessor(ms, typeGetter)

	// create observer
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		postprocessors,
		runner,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-sample-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	return tickers.NewSampleTicker(
		ratio,
		getter,
		observer,
		subscriber,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-sample-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)
}

func newFinalConditionalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn ocr2keepersv3.Runner,
	interval time.Duration,
	proposalQ ocr2keepers.ProposalQueue,
	builder ocr2keepers.PayloadBuilder,
	retryQ ocr2keepers.RetryQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "conditional-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "conditional-final-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "conditional-final-ineligible-postprocessor")),
	)
	// create observer that only pushes results to result stores. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	ticker := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](interval, observer, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return coordinatedProposalsTick{
			logger:    logger,
			builder:   builder,
			q:         proposalQ,
			utype:     ocr2keepers.ConditionTrigger,
			batchSize: RetryBatchSize,
		}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return ticker
}
