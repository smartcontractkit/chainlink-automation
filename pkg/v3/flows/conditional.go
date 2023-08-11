package flows

import (
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
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
	mStore  store.MetadataStore
	logger  *log.Logger
}

// NewConditionalEligibility ...
func NewConditionalEligibility(
	ratio Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	subscriber ocr2keepers.BlockSubscriber,
	builder ocr2keepers.PayloadBuilder,
	rs ResultStore,
	ms store.MetadataStore,
	rn ocr2keepersv3.Runner,
	logger *log.Logger,
) (*ConditionalEligibility, []service.Recoverable, error) {
	// TODO: add coordinator to preprocessor list
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{}

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	svc0 := newFinalConditionalFlow(preprocessors, rs, rn, time.Second, logger)

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	svc1, err := newSampleProposalFlow(preprocessors, ratio, getter, subscriber, ms, rn, logger)
	if err != nil {
		return nil, nil, err
	}

	return &ConditionalEligibility{
		mStore:  ms,
		builder: builder,
		logger:  logger,
	}, []service.Recoverable{svc0, svc1}, err
}

func (flow *ConditionalEligibility) ProcessOutcome(outcome ocr2keepersv3.AutomationOutcome) error {
	// TODO: Refactor into coordinatedProposals Flow
	/*
		samples, err := outcome.UpkeepIdentifiers()
		if err != nil {
			if errors.Is(err, ocr2keepersv3.ErrWrongDataType) {
				return err
			}

			flow.logger.Printf("%s", err)

			return nil
		}

		if len(samples) == 0 {
			return nil
		}

		// limit timeout to get all proposal data
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// merge block number and recoverables
		for _, sample := range samples {
			proposal := ocr2keepers.CoordinatedProposal{
				UpkeepID: sample,
			}

			payloads, err := flow.builder.BuildPayloads(ctx, proposal)
			if err != nil {
				flow.logger.Printf("error encountered when building payload")
				continue
			}
			if len(payloads) == 0 {
				flow.logger.Printf("did not get any results when building payload")
				continue
			}
			payload := payloads[0]

			// pass to recoverer
			if err := flow.final.Retry(ocr2keepers.CheckResult{
				UpkeepID: payload.UpkeepID,
				Trigger:  payload.Trigger,
			}); err != nil {
				continue
			}
		}

		// reset samples in metadata
		flow.mStore.Set(store.ProposalSampleMetadata, []ocr2keepers.UpkeepIdentifier{})
	*/
	return nil
}

func newSampleProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ratio Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	subscriber ocr2keepers.BlockSubscriber,
	ms store.MetadataStore,
	rn ocr2keepersv3.Runner,
	logger *log.Logger,
) (service.Recoverable, error) {
	// create a metadata store postprocessor
	pp := postprocessors.NewAddSamplesToMetadataStorePostprocessor(ms)

	// create observer
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		pp,
		rn,
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
	logger *log.Logger,
) service.Recoverable {
	// create observer that only pushes results to result store. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		postprocessors.NewEligiblePostProcessor(rs, log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-eligible-postprocessor]", telemetry.ServiceName), telemetry.LogPkgStdFlags)),
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create schedule ticker to manage retry interval
	ticker := tickers.NewBasicTicker[ocr2keepers.UpkeepPayload](
		interval,
		observer,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	return ticker
}
