package flows

import (
	"context"
	"fmt"
	"log"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
}

type UpkeepProvider interface {
	GetActiveUpkeeps(context.Context, ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error)
}

// ConditionalEligibility is a flow controller that surfaces conditional upkeeps
type ConditionalEligibility struct{}

// NewConditionalEligibility ...
func NewConditionalEligibility(
	ratio Ratio,
	getter UpkeepProvider,
	subscriber tickers.BlockSubscriber,
	rs ResultStore,
	ms MetadataStore,
	rn Runner,
	logger *log.Logger,
) (*ConditionalEligibility, []service.Recoverable, error) {
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{}

	// TODO: runner flow

	// proposal flow
	svc0, err := newSampleProposalFlow(preprocessors, ratio, getter, subscriber, ms, rn, logger)
	if err != nil {
		return nil, nil, err
	}

	return &ConditionalEligibility{}, []service.Recoverable{svc0}, err
}

func (flow *ConditionalEligibility) ProcessOutcome(_ ocr2keepersv3.AutomationOutcome) error {
	// TODO: complete reading of the outcome

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
	observer := ocr2keepersv3.NewRunnableObserver(preprocessors, nil, rn, ObservationProcessLimit)

	// create a metadata store postprocessor

	ticker, err := tickers.NewSampleTicker(
		ratio,
		getter,
		observer,
		subscriber,
		log.New(logger.Writer(), fmt.Sprintf("[%s | conditional-sample-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)
	if err != nil {
		return nil, err
	}

	return ticker, nil
}
