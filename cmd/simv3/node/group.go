package node

import (
	"io"
	"log"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	simio "github.com/smartcontractkit/ocr2keepers/cmd/simv3/io"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/net"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/ocr"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulate/upkeep"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
)

type GroupConfig struct {
	Runbook    config.RunBook
	Digester   types.OffchainConfigDigester
	Upkeeps    []chain.SimulatedUpkeep
	Collectors []telemetry.Collector
	Logger     *log.Logger
}

type Group struct {
	conf        config.RunBook
	nodes       map[string]*Simulator
	network     *net.SimulatedNetwork
	digester    types.OffchainConfigDigester
	blockSrc    *chain.BlockBroadcaster
	transmitter *ocr.OCR3TransmitLoader
	confLoader  *ocr.OCR3ConfigLoader
	upkeeps     []chain.SimulatedUpkeep
	monitor     commontypes.MonitoringEndpoint
	collectors  []telemetry.Collector
	logger      *log.Logger
}

func NewGroup(conf GroupConfig) *Group {
	// TODO: monitor data is not text so not sure what to do with this yet
	monitor := simio.NewMonitorToWriter(io.Discard)

	lTransmit := ocr.NewOCR3TransmitLoader(conf.Runbook, conf.Logger)
	lOCR3Config := ocr.NewOCR3ConfigLoader(conf.Runbook, conf.Digester, conf.Logger)

	lUpkeep, err := upkeep.NewUpkeepConfigLoader(conf.Runbook)
	if err != nil {
		panic(err)
	}

	lLogTriggers, err := upkeep.NewLogTriggerLoader(conf.Runbook)
	if err != nil {
		panic(err)
	}

	loaders := []chain.BlockLoaderFunc{
		lTransmit.Load,
		lOCR3Config.Load,
		lUpkeep.Load,
		lLogTriggers.Load,
	}

	return &Group{
		conf:        conf.Runbook,
		nodes:       make(map[string]*Simulator),
		network:     net.NewSimulatedNetwork(conf.Runbook.AvgNetworkLatency.Value()),
		digester:    conf.Digester,
		blockSrc:    chain.NewBlockBroadcaster(conf.Runbook.BlockCadence, conf.Runbook.RPCDetail.MaxBlockDelay, conf.Logger, loaders...),
		transmitter: lTransmit,
		confLoader:  lOCR3Config,
		upkeeps:     conf.Upkeeps,
		monitor:     monitor,
		collectors:  conf.Collectors,
		logger:      conf.Logger,
	}
}
