package node

import (
	"io"
	"log"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	simio "github.com/smartcontractkit/ocr2keepers/tools/simulator/io"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/loader"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/net"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/telemetry"
)

type GroupConfig struct {
	SimulationPlan config.SimulationPlan
	Digester       types.OffchainConfigDigester
	Upkeeps        []chain.SimulatedUpkeep
	Collectors     []telemetry.Collector
	Logger         *log.Logger
}

type Group struct {
	conf        config.SimulationPlan
	nodes       map[string]*Simulator
	network     *net.SimulatedNetwork
	digester    types.OffchainConfigDigester
	blockSrc    *chain.BlockBroadcaster
	transmitter *loader.OCR3TransmitLoader
	confLoader  *loader.OCR3ConfigLoader
	upkeeps     []chain.SimulatedUpkeep
	monitor     commontypes.MonitoringEndpoint
	collectors  []telemetry.Collector
	logger      *log.Logger
}

func NewGroup(conf GroupConfig, progress *telemetry.ProgressTelemetry) (*Group, error) {
	lTransmit, err := loader.NewOCR3TransmitLoader(conf.SimulationPlan, progress, conf.Logger)
	if err != nil {
		return nil, err
	}

	lOCR3Config := loader.NewOCR3ConfigLoader(conf.SimulationPlan, progress, conf.Digester, conf.Logger)

	lUpkeep, err := loader.NewUpkeepConfigLoader(conf.SimulationPlan, progress)
	if err != nil {
		return nil, err
	}

	lLogTriggers, err := loader.NewLogTriggerLoader(conf.SimulationPlan, progress)
	if err != nil {
		return nil, err
	}

	loaders := []chain.BlockLoaderFunc{
		lTransmit.Load,
		lOCR3Config.Load,
		lUpkeep.Load,
		lLogTriggers.Load,
	}

	return &Group{
		conf:        conf.SimulationPlan,
		nodes:       make(map[string]*Simulator),
		network:     net.NewSimulatedNetwork(conf.SimulationPlan.Network.MaxLatency.Value()),
		digester:    conf.Digester,
		blockSrc:    chain.NewBlockBroadcaster(conf.SimulationPlan.Blocks, conf.SimulationPlan.RPC.MaxBlockDelay, conf.Logger, progress, loaders...),
		transmitter: lTransmit,
		confLoader:  lOCR3Config,
		upkeeps:     conf.Upkeeps,
		monitor:     simio.NewMonitorToWriter(io.Discard), // monitor data is not text so not sure what to do with this yet
		collectors:  conf.Collectors,
		logger:      conf.Logger,
	}, nil
}
