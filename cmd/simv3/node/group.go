package node

import (
	"io"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/blocks"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	simio "github.com/smartcontractkit/ocr2keepers/cmd/simv3/io"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulators"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

type GroupConfig struct {
	Digester      types.OffchainConfigDigester
	Cadence       config.Blocks
	Encoder       plugin.Encoder
	Upkeeps       []simulators.SimulatedUpkeep
	ConfigEvents  []config.ConfigEvent
	RPCConfig     config.RPC
	AvgNetLatency time.Duration
	Collectors    []telemetry.Collector
	Logger        *log.Logger
}

type Group struct {
	nodes       map[string]*Simulator
	network     *simulators.SimulatedNetwork
	digester    types.OffchainConfigDigester
	blockSrc    *blocks.BlockBroadcaster
	encoder     plugin.Encoder
	transmitter *blocks.TransmitLoader
	confLoader  *blocks.ConfigLoader
	upkeeps     []simulators.SimulatedUpkeep
	monitor     commontypes.MonitoringEndpoint
	rpcConf     config.RPC
	collectors  []telemetry.Collector
	logger      *log.Logger
}

func NewGroup(conf GroupConfig) *Group {
	transmit := blocks.NewTransmitLoader()
	confLoad := blocks.NewConfigLoader(conf.ConfigEvents, conf.Digester)

	// TODO: monitor data is not text so not sure what to do with this yet
	monitor := simio.NewMonitorToWriter(io.Discard)

	return &Group{
		nodes:       make(map[string]*Simulator),
		network:     simulators.NewSimulatedNetwork(conf.AvgNetLatency),
		digester:    conf.Digester,
		blockSrc:    blocks.NewBlockBroadcaster(conf.Cadence, conf.RPCConfig.MaxBlockDelay, transmit, confLoad),
		encoder:     conf.Encoder,
		transmitter: transmit,
		confLoader:  confLoad,
		upkeeps:     conf.Upkeeps,
		monitor:     monitor,
		rpcConf:     conf.RPCConfig,
		collectors:  conf.Collectors,
		logger:      conf.Logger,
	}
}
