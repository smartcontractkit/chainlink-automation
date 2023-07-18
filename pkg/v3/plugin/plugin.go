package plugin

import (
	"log"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/build"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/prebuild"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/instructions"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func newPlugin(
	logProvider flows.LogEventProvider,
	events coordinator.EventProvider,
	blockSource tickers.BlockSubscriber,
	rp flows.RecoverableProvider,
	encoder Encoder,
	runnable runner.Runnable,
	rConf runner.RunnerConfig,
	conf config.OffchainConfig,
	logger *log.Logger,
) (ocr3types.ReportingPlugin[AutomationReportInfo], error) {
	blockTicker, err := tickers.NewBlockTicker(blockSource)
	if err != nil {
		return nil, err
	}

	// create the value stores
	rs := resultstore.New(logger)
	ms := store.NewMetadata(blockTicker)
	is := instructions.NewStore()

	// create a new runner instance
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
		ms,
		rn,
		logProvider,
		rp,
		flows.LogCheckInterval,
		flows.RecoveryCheckInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			// TODO: provide configuration inputs
			tickers.ScheduleTickerWithDefaults,
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			func(c *tickers.ScheduleTickerConfig) {
				// TODO: provide configuration inputs
				c.SendDelay = 5 * time.Minute
				c.MaxSendDuration = 24 * time.Hour
			},
		},
	)

	// create the event coordinator
	coord := coordinator.NewReportCoordinator(events, conf, logger)

	// create service recoverers to provide panic recovery on dependent services
	allSvcs := append(svcs, []service.Recoverable{rs, ms, coord, rn, blockTicker}...)
	recoverSvcs := []service.Recoverable{}

	for i := range allSvcs {
		recoverSvcs = append(recoverSvcs, service.NewRecoverer(allSvcs[i], logger))
	}

	// pass the eligibility flow to the plugin as a hook since it uses outcome
	// data
	plugin := &ocr3Plugin{
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
			prebuild.NewRemoveFromStaging(rs, logger).RunHook,
			prebuild.NewCoordinateBlockHook(is, ms).RunHook,
		},
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation) error{
			build.NewAddFromStaging(rs, logger).RunHook,
			build.NewCoordinateBlockHook(is, ms).RunHook,
		},
		ReportEncoder: encoder,
		Coordinators:  []Coordinator{coord},
		Services:      recoverSvcs,
		Config:        conf,
		Logger:        logger,
	}

	plugin.startServices()

	return plugin, nil
}
