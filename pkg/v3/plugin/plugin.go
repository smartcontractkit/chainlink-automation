package plugin

import (
	"context"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func New(
	ctx context.Context,
	logLookup ocr2keepersv3.PreProcessor,
	logger *log.Logger,
) (ocr3types.OCR3Plugin[string], error) {
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
	plugin := &ocr3Plugin[string]{
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
			hooks.NewPrebuildHookRemoveFromStaging(rs).RunHook,
		},
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation, ocr2keepersv3.InstructionStore, ocr2keepersv3.SamplingStore, ocr2keepersv3.ResultStore) error{
			hooks.NewBuildHookAddFromStaging().RunHook,
		},
		ResultSource: rs,
	}

	// log trigger eligibility flow contains numerous services to start
	if err := ltFlow.Start(ctx); err != nil {
		return nil, err
	}

	// result store contains numerous service to start
	if err := rs.Start(ctx); err != nil {
		return nil, err
	}

	return plugin, nil
}
