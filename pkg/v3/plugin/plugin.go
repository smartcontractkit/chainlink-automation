package plugin

import (
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func newPlugin[RI any](
	logLookup ocr2keepersv3.PreProcessor,
	encoder Encoder,
	logger *log.Logger,
) (ocr3types.OCR3Plugin[RI], error) {
	var (
		rn ocr2keepersv3.Runner
	)

	rs := resultstore.New(logger)

	// initialize the log trigger eligibility flow
	ltFlow, svcs := flows.NewLogTriggerEligibility(
		logLookup,
		rs,
		rn,
		logger,
		tickers.RetryWithDefaults,
	)

	allSvcs := append(svcs, rs)
	recoverSvcs := []service.Recoverable{}

	for i := range allSvcs {
		recoverSvcs = append(recoverSvcs, service.NewRecoverer(allSvcs[i], logger))
	}

	// pass the eligibility flow to the plugin as a hook since it uses outcome
	// data
	plugin := &ocr3Plugin[RI]{
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
			hooks.NewPrebuildHookRemoveFromStaging(rs).RunHook,
		},
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation, ocr2keepersv3.InstructionStore, ocr2keepersv3.MetadataStore, ocr2keepersv3.ResultStore) error{
			hooks.NewBuildHookAddFromStaging().RunHook,
		},
		ResultSource:       rs,
		ReportEncoder:      encoder,
		Services:           recoverSvcs,
		ReportGasLimit:     5_000_000,
		MaxUpkeepBatchSize: 1,
	}

	plugin.startServices()

	return plugin, nil
}
