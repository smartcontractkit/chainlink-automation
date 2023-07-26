package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

//go:generate mockery --name Ratio --structname MockRatio --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename ratio.generated.go
type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
}

//go:generate mockery --name UpkeepProvider --structname MockUpkeepProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename upkeepprovider.generated.go
type UpkeepProvider interface {
	GetActiveUpkeeps(context.Context, ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error)
}

// ConditionalEligibility is a flow controller that surfaces conditional upkeeps
type ConditionalEligibility struct {
	builder PayloadBuilder
	final   Retryer
	logger  *log.Logger
}

// NewConditionalEligibility ...
func NewConditionalEligibility(
	ratio Ratio,
	getter UpkeepProvider,
	subscriber tickers.BlockSubscriber,
	builder PayloadBuilder,
	rs ResultStore,
	ms MetadataStore,
	rn Runner,
	logger *log.Logger,
) (*ConditionalEligibility, []service.Recoverable, error) {
	// TODO: add coordinator to preprocessor list
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{}

	// runs full check pipeline on a coordinated block with coordinated upkeeps
	svc0, point := newFinalConditionalFlow(preprocessors, rs, rn, time.Second, logger)

	// the sampling proposal flow takes random samples of active upkeeps, checks
	// them and surfaces the ids if the items are eligible
	svc1, err := newSampleProposalFlow(preprocessors, ratio, getter, subscriber, ms, rn, logger)
	if err != nil {
		return nil, nil, err
	}

	return &ConditionalEligibility{
		builder: builder,
		final:   point,
		logger:  logger,
	}, []service.Recoverable{svc0, svc1}, err
}

func (flow *ConditionalEligibility) ProcessOutcome(outcome ocr2keepersv3.AutomationOutcome) error {
	var ok bool

	rawSamples, ok := outcome.Metadata[ocr2keepersv3.CoordinatedSamplesProposalKey]
	if !ok {
		flow.logger.Printf("no proposed samples found in outcome")

		return nil
	}

	samples, ok := rawSamples.([]ocr2keepers.UpkeepIdentifier)
	if !ok {
		return fmt.Errorf("%w: coordinated proposals are not of type `UpkeepIdentifier`", ErrWrongDataType)
	}

	// get latest coordinated block
	// by checking latest outcome first and then looping through the history
	var (
		rawBlock       interface{}
		blockAvailable bool
		block          ocr2keepers.BlockKey
	)

	if rawBlock, ok = outcome.Metadata[ocr2keepersv3.CoordinatedBlockOutcomeKey]; !ok {
		for _, h := range historyFromRingBuffer(outcome.History, outcome.NextIdx) {
			if rawBlock, ok = h.Metadata[ocr2keepersv3.CoordinatedBlockOutcomeKey]; !ok {
				continue
			}

			blockAvailable = true

			break
		}
	} else {
		blockAvailable = true
	}

	// we have proposals but a latest block isn't available
	if !blockAvailable {
		return ErrBlockNotAvailable
	}

	if block, ok = rawBlock.(ocr2keepers.BlockKey); !ok {
		return fmt.Errorf("%w: coordinated block value not of type `BlockKey`", ErrWrongDataType)
	}

	// limit timeout to get all proposal data
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// merge block number and recoverables
	for _, sample := range samples {
		proposal := ocr2keepers.CoordinatedProposal{
			UpkeepID: sample,
			Block:    block,
		}

		payload, err := flow.builder.BuildPayload(ctx, proposal)
		if err != nil {
			flow.logger.Printf("error encountered when building payload")

			continue
		}

		// pass to recoverer
		if err := flow.final.Retry(ocr2keepers.CheckResult{
			Payload: payload,
		}); err != nil {
			continue
		}
	}

	return nil
}

func newSampleProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ratio Ratio,
	getter UpkeepProvider,
	subscriber tickers.BlockSubscriber,
	ms MetadataStore,
	rn Runner,
	logger *log.Logger,
) (service.Recoverable, error) {
	// create observer
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		new(emptyPP),
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-sample-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create a metadata store postprocessor
	pp := postprocessors.NewAddSamplesToMetadataStorePostprocessor(ms)

	// create observer
	observer := ocr2keepersv3.NewRunnableObserver(preprocessors, pp, rn, ObservationProcessLimit)

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
	rn Runner,
	interval time.Duration,
	logger *log.Logger,
) (service.Recoverable, Retryer) {
	// create observer that only pushes results to result store. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		postprocessors.NewEligiblePostProcessor(rs),
		rn,
		ObservationProcessLimit,
	)

	// create schedule ticker to manage retry interval
	ticker := tickers.NewBasicTicker[ocr2keepers.UpkeepPayload](
		interval,
		observer,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// wrap schedule ticker as a Retryer
	// this provides a common interface for processors and hooks
	retryer := &basicRetryer{ticker: ticker}

	return ticker, retryer
}
