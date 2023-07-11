package node

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	simio "github.com/smartcontractkit/ocr2keepers/cmd/simv3/io"
	pluginconfig "github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulators"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
)

func (g *Group) Add(maxWorkers int, maxQueueSize int) {
	net := g.network.NewFactory()

	var rpcTel *telemetry.RPCCollector
	var logTel *telemetry.NodeLogCollector
	var ctrTel *telemetry.ContractEventCollector

	for _, col := range g.collectors {
		switch cT := col.(type) {
		case *telemetry.RPCCollector:
			rpcTel = cT
		case *telemetry.NodeLogCollector:
			logTel = cT
		case *telemetry.ContractEventCollector:
			ctrTel = cT
		}
	}

	offchainRing, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)
	if err != nil {
		panic(err)
	}

	onchainRing, err := config.NewEVMKeyring(rand.Reader)
	if err != nil {
		panic(err)
	}

	_ = logTel.AddNode(net.PeerID())
	_ = rpcTel.AddNode(net.PeerID())
	_ = ctrTel.AddNode(net.PeerID())

	// general logger
	var slogger *simio.SimpleLogger
	var cLogger *log.Logger
	var gLogger *log.Logger

	if logTel != nil {
		slogger = simio.NewSimpleLogger(logTel.GeneralLog(net.PeerID()), simio.Debug)
		cLogger = log.New(logTel.ContractLog(net.PeerID()), "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		gLogger = log.New(logTel.GeneralLog(net.PeerID()), "[general] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	} else {
		slogger = simio.NewSimpleLogger(io.Discard, simio.Critical)
		cLogger = log.New(io.Discard, "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		gLogger = log.New(io.Discard, "[general] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	ct := simulators.NewSimulatedContract(
		g.blockSrc,
		g.digester,
		g.upkeeps,
		g.encoder,
		g.transmitter,
		g.rpcConf.AverageLatency,
		onchainRing.PKString(),
		g.rpcConf.ErrorRate,
		g.rpcConf.RateLimitThreshold,
		ctrTel.ContractEventCollectorNode(net.PeerID()),
		rpcTel.RPCCollectorNode(net.PeerID()),
		cLogger)
	db := simulators.NewSimulatedDatabase()

	runr, _ := runner.NewRunner(
		gLogger,
		ct,
		runner.RunnerConfig{
			Workers:           maxWorkers,
			WorkerQueueLength: maxQueueSize,
			CacheExpire:       pluginconfig.DefaultCacheExpiration,
			CacheClean:        pluginconfig.DefaultCacheClearInterval,
		},
	)

	dConfig := plugin.DelegateConfig{
		BinaryNetworkEndpointFactory: net,
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		ContractConfigTracker:        ct,
		ContractTransmitter:          ct,
		KeepersDatabase:              db,
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  time.Second,
			ContractConfigConfirmations:        1,
			SkipContractConfigConfirmations:    false,
			ContractConfigTrackerPollInterval:  15 * time.Second,
			ContractTransmitterTransmitTimeout: time.Second,
			DatabaseTimeout:                    time.Second,
			DevelopmentMode:                    "",
		},
		LogProvider:            ct,
		EventProvider:          ct,
		Runnable:               runr,
		Encoder:                g.encoder,
		Logger:                 slogger,
		MonitoringEndpoint:     g.monitor,
		OffchainConfigDigester: g.digester,
		OffchainKeyring:        offchainRing,
		OnchainKeyring:         onchainRing,
		MaxServiceWorkers:      maxWorkers,
		ServiceQueueLength:     maxQueueSize,
	}

	service, err := plugin.NewDelegate(dConfig)
	if err != nil {
		panic(err)
	}

	g.nodes[net.PeerID()] = &Simulator{
		Service:  service,
		Contract: ct,
	}

	g.logger.Println("starting new node")

	_ = service.Start(context.Background())
	ct.Start()

	g.confLoader.AddSigner(net.PeerID(), onchainRing, offchainRing)
}

func (g *Group) Start(ctx context.Context, count int, maxWorkers int, maxQueueSize int) error {
	var err error

	for i := 0; i < count; i++ {
		g.Add(maxWorkers, maxQueueSize)
	}

	log.Print("starting simulation")
	select {
	case <-g.blockSrc.Start():
		err = fmt.Errorf("block duration ended")
	case <-ctx.Done():
		g.blockSrc.Stop()
		err = fmt.Errorf("SIGTERM event stopping process")
	}

	g.ReportResults()

	for id, node := range g.nodes {
		if err := node.Service.Close(); err != nil {
			log.Printf("error encountered while attempting to close down node '%s': %s", id, err)
		}
		node.Contract.Stop()
	}

	for _, col := range g.collectors {
		col.Close()
	}

	return err
}
