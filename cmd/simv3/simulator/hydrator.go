package simulator

import (
	"log"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/db"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/ocr"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/upkeep"
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
	logger *log.Logger,
) error {
	listener := chain.NewListener(blocks, logger)
	active := upkeep.NewActiveTracker(listener, logger)

	triggered := upkeep.NewLogTriggerTracker(listener, active, logger)
	source := upkeep.NewSource(listener, active, triggered, DefaultLookbackBlocks, logger)

	config.ContractConfigTracker = ocr.NewOCR3ConfigTracker(listener, logger)
	config.ContractTransmitter = ocr.NewOCR3Transmitter(name, transmitter)
	config.KeepersDatabase = db.NewSimulatedOCR3Database()

	config.LogProvider = source
	config.EventProvider = ocr.NewReportTracker(listener, logger)
	config.Runnable = upkeep.NewCheckPipeline(listener, active, logger)

	config.Encoder = source.Util
	config.BlockSubscriber = chain.NewBlockHistoryTracker(listener, logger)
	config.RecoverableProvider = source

	config.PayloadBuilder = source.Util
	config.UpkeepProvider = source
	config.UpkeepStateUpdater = db.NewUpkeepStateDatabase()

	config.UpkeepTypeGetter = source.Util.GetType
	config.WorkIDGenerator = source.Util.GenerateWorkID

	return nil
}
