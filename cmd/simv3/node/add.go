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
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator"
	pluginconfig "github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
)

func (g *Group) Add(maxWorkers int, maxQueueSize int) {
	simNet := g.network.NewFactory()

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

	_ = logTel.AddNode(simNet.PeerID())
	_ = rpcTel.AddNode(simNet.PeerID())
	_ = ctrTel.AddNode(simNet.PeerID())

	// general logger
	var slogger *simio.SimpleLogger
	var cLogger *log.Logger
	var gLogger *log.Logger

	if logTel != nil {
		slogger = simio.NewSimpleLogger(logTel.GeneralLog(simNet.PeerID()), simio.Debug)
		cLogger = log.New(logTel.ContractLog(simNet.PeerID()), "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		gLogger = log.New(logTel.GeneralLog(simNet.PeerID()), "[general] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	} else {
		slogger = simio.NewSimpleLogger(io.Discard, simio.Critical)
		cLogger = log.New(io.Discard, "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		gLogger = log.New(io.Discard, "[general] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	}

	dConfig := plugin.DelegateConfig{
		BinaryNetworkEndpointFactory: simNet,
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  time.Second,
			ContractConfigConfirmations:        1,
			SkipContractConfigConfirmations:    false,
			ContractConfigTrackerPollInterval:  15 * time.Second,
			ContractTransmitterTransmitTimeout: time.Second,
			DatabaseTimeout:                    time.Second,
			DevelopmentMode:                    "",
		},
		Logger:                 slogger,
		MonitoringEndpoint:     g.monitor,
		OffchainConfigDigester: g.digester,
		OffchainKeyring:        offchainRing,
		OnchainKeyring:         onchainRing,
		MaxServiceWorkers:      maxWorkers,
		ServiceQueueLength:     maxQueueSize,
	}

	_ = simulator.HydrateConfig(
		onchainRing.PKString(),
		&dConfig,
		g.blockSrc,
		g.transmitter,
		g.conf,
		rpcTel.RPCCollectorNode(simNet.PeerID()),
		ctrTel.ContractEventCollectorNode(simNet.PeerID()),
		cLogger,
	)

	runr, _ := runner.NewRunner(
		gLogger,
		dConfig.Runnable,
		runner.RunnerConfig{
			Workers:           maxWorkers,
			WorkerQueueLength: maxQueueSize,
			CacheExpire:       pluginconfig.DefaultCacheExpiration,
			CacheClean:        pluginconfig.DefaultCacheClearInterval,
		},
	)

	dConfig.Runnable = runr

	service, err := plugin.NewDelegate(dConfig)
	if err != nil {
		panic(err)
	}

	g.nodes[simNet.PeerID()] = &Simulator{
		Service: service,
	}

	g.logger.Println("starting new node")

	_ = service.Start(context.Background())

	g.confLoader.AddSigner(simNet.PeerID(), onchainRing, offchainRing)
}

func (g *Group) Start(ctx context.Context, count int, maxWorkers int, maxQueueSize int) error {
	var err error

	for i := 0; i < count; i++ {
		g.Add(maxWorkers, maxQueueSize)
	}

	g.logger.Print("starting simulation")
	select {
	case <-g.blockSrc.Start():
		err = fmt.Errorf("block duration ended")
	case <-ctx.Done():
		g.blockSrc.Stop()
		err = fmt.Errorf("SIGTERM event stopping process")
	}

	g.WriteTransmitChart()
	g.ReportResults()

	for id, node := range g.nodes {
		if err := node.Service.Close(); err != nil {
			log.Printf("error encountered while attempting to close down node '%s': %s", id, err)
		}
	}

	for _, col := range g.collectors {
		col.Close()
	}

	return err
}
