package plugin

import (
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
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
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func newPlugin(
	digest types.ConfigDigest,
	logProvider flows.LogEventProvider,
	events coordinator.EventProvider,
	blockSource tickers.BlockSubscriber,
	rp flows.RecoverableProvider,
	builder flows.PayloadBuilder,
	ratio flows.Ratio,
	getter flows.UpkeepProvider,
	encoder Encoder,
	runnable runner.Runnable,
	rConf runner.RunnerConfig,
	conf config.OffchainConfig,
	f int,
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

	// on plugin startup, begin broadcasting that block coordination should
	// happen immediately
	is.Set(instructions.ShouldCoordinateBlock)

	// add recovery cache to metadata store with 24hr timeout
	ms.Set(store.ProposalRecoveryMetadata, util.NewCache[ocr2keepers.CoordinatedProposal](24*time.Hour))

	// create a new runner instance
	rn, err := runner.NewRunner(
		logger,
		runnable,
		rConf,
	)
	if err != nil {
		return nil, err
	}

	// create the event coordinator
	coord := coordinator.NewReportCoordinator(events, conf, logger)

	// initialize the log trigger eligibility flow
	ltFlow, svcs := flows.NewLogTriggerEligibility(
		coord,
		rs,
		ms,
		rn,
		logProvider,
		rp,
		builder,
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

	// create service recoverers to provide panic recovery on dependent services
	allSvcs := append(svcs, []service.Recoverable{rs, ms, coord, rn, blockTicker}...)

	cFlow, svcs, err := flows.NewConditionalEligibility(ratio, getter, blockSource, builder, rs, ms, rn, logger)
	if err != nil {
		return nil, err
	}

	allSvcs = append(allSvcs, svcs...)

	recoverSvcs := []service.Recoverable{}

	for i := range allSvcs {
		recoverSvcs = append(recoverSvcs, service.NewRecoverer(allSvcs[i], logger))
	}

	// pass the eligibility flow to the plugin as a hook since it uses outcome
	// data
	plugin := &ocr3Plugin{
		ConfigDigest: digest,
		PrebuildHooks: []func(ocr2keepersv3.AutomationOutcome) error{
			ltFlow.ProcessOutcome,
			cFlow.ProcessOutcome,
			prebuild.NewRemoveFromStaging(rs, logger).RunHook,
			prebuild.NewCoordinateBlockHook(is, ms).RunHook,
		},
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation) error{
			build.NewAddFromStaging(rs, logger).RunHook,
			//build.NewCoordinateBlockHook(is, ms).RunHook,
			//build.NewAddFromRecoveryHook(ms).RunHook,
			//build.NewAddFromSamplesHook(ms).RunHook,
		},
		ReportEncoder: encoder,
		Coordinators:  []Coordinator{coord},
		Services:      recoverSvcs,
		Config:        conf,
		F:             f,
		Logger:        log.New(logger.Writer(), fmt.Sprintf("[%s | plugin]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}

	plugin.startServices()

	return plugin, nil
}
