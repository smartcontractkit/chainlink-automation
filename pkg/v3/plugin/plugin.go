package plugin

import (
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin/hooks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/stores"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func newPlugin(
	digest types.ConfigDigest,
	logProvider ocr2keepers.LogEventProvider,
	events ocr2keepers.TransmitEventProvider,
	blockSource ocr2keepers.BlockSubscriber,
	recoverablesProvider ocr2keepers.RecoverableProvider,
	builder ocr2keepers.PayloadBuilder,
	ratio ocr2keepers.Ratio,
	getter ocr2keepers.ConditionalUpkeepProvider,
	encoder ocr2keepers.Encoder,
	upkeepTypeGetter ocr2keepers.UpkeepTypeGetter,
	upkeepStateUpdater ocr2keepers.UpkeepStateUpdater,
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
	resultStore := stores.New(logger)
	metadataStore := stores.NewMetadataStore(blockTicker, upkeepTypeGetter)

	// create a new runner instance
	runner, err := runner.NewRunner(
		logger,
		runnable,
		rConf,
	)
	if err != nil {
		return nil, err
	}

	// create the event coordinator
	coord := coordinator.NewCoordinator(events, upkeepTypeGetter, conf, logger)

	retryQ := stores.NewRetryQueue(logger)

	retrySvc := flows.NewRetryFlow(coord, resultStore, runner, retryQ, 5*time.Second, upkeepStateUpdater, logger)

	proposalQ := stores.NewProposalQueue(upkeepTypeGetter)

	// initialize the log trigger eligibility flow
	svcs := flows.LogTriggerFlows(
		coord,
		resultStore,
		metadataStore,
		runner,
		logProvider,
		recoverablesProvider,
		builder,
		flows.LogCheckInterval,
		flows.RecoveryCheckInterval,
		retryQ,
		proposalQ,
		upkeepStateUpdater,
		upkeepTypeGetter,
		logger,
	)

	// create service recoverers to provide panic recovery on dependent services
	allSvcs := append(svcs, []service.Recoverable{retrySvc, resultStore, metadataStore, coord, runner, blockTicker}...)

	svcs, err = flows.ConditionalTriggerFlows(
		coord,
		ratio,
		getter,
		blockSource,
		builder,
		resultStore,
		metadataStore,
		runner,
		proposalQ,
		retryQ,
		upkeepStateUpdater,
		upkeepTypeGetter,
		logger,
	)
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
		ConfigDigest:                digest,
		ReportEncoder:               encoder,
		Coordinator:                 coord,
		RemoveFromStagingHook:       hooks.NewRemoveFromStagingHook(resultStore, logger),
		RemoveFromMetadataHook:      hooks.NewRemoveFromMetadataHook(metadataStore, upkeepTypeGetter, logger),
		AddToProposalQHook:          hooks.NewAddToProposalQHook(proposalQ, logger),
		AddBlockHistoryHook:         hooks.NewAddBlockHistoryHook(metadataStore, logger),
		AddFromStagingHook:          hooks.NewAddFromStagingHook(resultStore, coord, logger),
		AddConditionalSamplesHook:   hooks.NewAddConditionalSamplesHook(metadataStore, coord, logger),
		AddLogRecoveryProposalsHook: hooks.NewAddLogRecoveryProposalsHook(metadataStore, coord, logger),
		Services:                    recoverSvcs,
		Config:                      conf,
		F:                           f,
		Logger:                      log.New(logger.Writer(), fmt.Sprintf("[%s | plugin]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}

	plugin.startServices()

	return plugin, nil
}
