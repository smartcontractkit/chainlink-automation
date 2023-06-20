package plugin

import (
	"context"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type service interface {
	Start(context.Context) error
	Close() error
}

func newPlugin[RI any](
	logLookup ocr2keepersv3.PreProcessor,
	logger *log.Logger,
) (ocr3types.OCR3Plugin[RI], []service, error) {
	var (
		rn ocr2keepersv3.Runner
	)

	rs := resultstore.New(logger)

	// initialize the log trigger eligibility flow
	ltFlow := flows.NewLogTriggerEligibility(
		logLookup,
		rs,
		rn,
		logger,
		tickers.RetryWithDefaults,
	)

	// pass the eligibility flow to the plugin as a hook since it uses outcome
	// data
	plugin := &ocr3Plugin[RI]{
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
		},
		ResultSource: rs,
	}

	return plugin, []service{ltFlow, rs}, nil
}
