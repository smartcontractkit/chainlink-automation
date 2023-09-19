package simulate

import (
	"log"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/db"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/net"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/ocr"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/upkeep"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

const (
	DefaultLookbackBlocks = 100
)

func HydrateConfig(
	name string,
	config *plugin.DelegateConfig,
	blocks *chain.BlockBroadcaster,
	transmitter *ocr.OCR3TransmitLoader,
	conf config.RunBook,
	netTelemetry net.NetTelemetry,
	conTelemetry *telemetry.WrappedContractCollector,
	logger *log.Logger,
) error {
	listener := chain.NewListener(blocks, logger)
	active := upkeep.NewActiveTracker(listener, logger)
	performs := upkeep.NewPerformTracker(listener, logger)

	triggered := upkeep.NewLogTriggerTracker(listener, active, performs, logger)
	source := upkeep.NewSource(active, triggered, DefaultLookbackBlocks, logger)

	config.ContractConfigTracker = ocr.NewOCR3ConfigTracker(listener, logger)
	config.ContractTransmitter = ocr.NewOCR3Transmitter(name, transmitter)
	config.KeepersDatabase = db.NewSimulatedOCR3Database()

	config.LogProvider = source
	config.EventProvider = ocr.NewReportTracker(listener, logger)
	config.Runnable = upkeep.NewCheckPipeline(conf, active, performs, netTelemetry, conTelemetry, logger)

	config.Encoder = source.Util
	config.BlockSubscriber = chain.NewBlockHistoryTracker(listener, logger)
	config.RecoverableProvider = source

	config.PayloadBuilder = source
	config.UpkeepProvider = source
	config.UpkeepStateUpdater = db.NewUpkeepStateDatabase()

	config.UpkeepTypeGetter = source.Util.GetType
	config.WorkIDGenerator = source.Util.GenerateWorkID

	return nil
}
