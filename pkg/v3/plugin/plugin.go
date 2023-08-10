package plugin

import (
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/build"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/prebuild"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/resultstore"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/retryqueue"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func newPlugin(
	digest types.ConfigDigest,
	logProvider ocr2keepers.LogEventProvider,
	events ocr2keepers.TransmitEventProvider,
	blockSource ocr2keepers.BlockSubscriber,
	rp ocr2keepers.RecoverableProvider,
	builder ocr2keepers.PayloadBuilder,
	ratio flows.Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	encoder ocr2keepers.Encoder,
	upkeepTypeGetter ocr2keepers.UpkeepTypeGetter,
	runnable ocr2keepers.Runnable,
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
	coord := coordinator.NewCoordinator(events, upkeepTypeGetter, conf, logger)

	retryQ := retryqueue.NewRetryQueue(logger)

	retrySvc := flows.NewRetryFlow(coord, rs, rn, retryQ, 5*time.Second, logger)

	// initialize the log trigger eligibility flow
	_, svcs := flows.NewLogTriggerEligibility(
		coord,
		rs,
		ms,
		rn,
		logProvider,
		rp,
		builder,
		flows.LogCheckInterval,
		flows.RecoveryCheckInterval,
		retryQ,
		logger,
	)

	// create service recoverers to provide panic recovery on dependent services
	allSvcs := append(svcs, []service.Recoverable{retrySvc, rs, ms, coord, rn, blockTicker}...)

	_, svcs, err = flows.NewConditionalEligibility(ratio, getter, blockSource, builder, rs, ms, rn, logger)
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
			// TODO: Condense these two flow hooks into a single coordinatedOutcome flow hook
			//ltFlow.ProcessOutcome,
			//cFlow.ProcessOutcome,
			prebuild.NewRemoveFromStaging(rs, logger).RunHook,
		},
		// TODO: add coordinator in build hook, pass limit and randSrc
		BuildHooks: []func(*ocr2keepersv3.AutomationObservation) error{
			build.NewAddFromStaging(rs, logger).RunHook,
			// TODO: AUTO-4243 Finalize build hooks
			//build.NewAddFromRecoveryHook(ms).RunHook,
			//build.NewAddFromSamplesHook(ms).RunHook,
		},
		ReportEncoder: encoder,
		Coordinator:   coord,
		Services:      recoverSvcs,
		Config:        conf,
		F:             f,
		Logger:        log.New(logger.Writer(), fmt.Sprintf("[%s | plugin]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}

	plugin.startServices()

	return plugin, nil
}
