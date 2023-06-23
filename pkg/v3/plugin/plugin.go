package plugin

import (
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func newPlugin[RI any](
	logProvider flows.LogEventProvider,
	events coordinator.EventProvider,
	encoder Encoder,
	runnable runner.Runnable,
	rConf runner.RunnerConfig,
	conf config.OffchainConfig,
	logger *log.Logger,
) (ocr3types.OCR3Plugin[RI], error) {
	rs := resultstore.New(logger)
	rn, err := runner.NewRunner(
		logger,
		runnable,
		rConf,
	)
	if err != nil {
		return nil, err
	}

	// initialize the log trigger eligibility flow
	ltFlow, svcs := flows.NewLogTriggerEligibility(
		rs,
		rn,
		logProvider,
		logger,
		tickers.RetryWithDefaults,
	)

	// create the event coordinator
	coord := coordinator.NewReportCoordinator(events, conf, logger)

	// create service recoverers to provide panic recovery on dependent services
	allSvcs := append(svcs, []service.Recoverable{rs, coord, rn}...)
	recoverSvcs := []service.Recoverable{}

	for i := range allSvcs {
		recoverSvcs = append(recoverSvcs, service.NewRecoverer(allSvcs[i], logger))
	}

	// pass the eligibility flow to the plugin as a hook since it uses outcome
	// data
	plugin := &ocr3Plugin[RI]{
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
			hooks.NewPrebuildHookRemoveFromStaging(rs, logger).RunHook,
		},
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation) error{
			hooks.NewBuildHookAddFromStaging(rs, logger).RunHook,
		},
		ReportEncoder: encoder,
		Coordinator:   coord,
		Services:      recoverSvcs,
		Config:        conf,
		Logger:        logger,
	}

	plugin.startServices()

	return plugin, nil
}
