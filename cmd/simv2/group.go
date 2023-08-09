package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/blocks"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/simulators"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/telemetry"
	pluginconfig "github.com/smartcontractkit/ocr2keepers/pkg/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v2"
	"github.com/smartcontractkit/ocr2keepers/pkg/v2/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v2/observer/polling"
	"github.com/smartcontractkit/ocr2keepers/pkg/v2/runner"
)

type Closable interface {
	Close() error
}

type Stoppable interface {
	Stop()
}

type ActiveNode struct {
	Service  Closable
	Contract Stoppable
}

type NodeGroupConfig struct {
	Digester      types.OffchainConfigDigester
	Cadence       config.Blocks
	Encoder       fullEncoder
	Upkeeps       []simulators.SimulatedUpkeep
	ConfigEvents  []config.ConfigEvent
	RPCConfig     config.RPC
	AvgNetLatency time.Duration
	MonitorIO     io.Writer
	LogPath       string
	Logger        *log.Logger
	Collectors    []telemetry.Collector
}

type NodeGroup struct {
	nodes       map[string]*ActiveNode
	network     *simulators.SimulatedNetwork
	digester    types.OffchainConfigDigester
	blockSrc    *blocks.BlockBroadcaster
	encoder     fullEncoder
	transmitter *blocks.TransmitLoader
	confLoader  *blocks.ConfigLoader
	upkeeps     []simulators.SimulatedUpkeep
	monitor     commontypes.MonitoringEndpoint
	rpcConf     config.RPC
	logPath     string
	logger      *log.Logger
	collectors  []telemetry.Collector
}

func NewNodeGroup(conf NodeGroupConfig) *NodeGroup {
	transmit := blocks.NewTransmitLoader()
	confLoad := blocks.NewConfigLoader(conf.ConfigEvents, conf.Digester)

	// general monitoring collector
	monitor := NewMonitorToWriter(conf.MonitorIO)

	return &NodeGroup{
		nodes:       make(map[string]*ActiveNode),
		network:     simulators.NewSimulatedNetwork(conf.AvgNetLatency),
		digester:    conf.Digester,
		blockSrc:    blocks.NewBlockBroadcaster(conf.Cadence, conf.RPCConfig.MaxBlockDelay, transmit, confLoad),
		encoder:     conf.Encoder,
		transmitter: transmit,
		confLoader:  confLoad,
		upkeeps:     conf.Upkeeps,
		monitor:     monitor,
		rpcConf:     conf.RPCConfig,
		logPath:     conf.LogPath,
		logger:      conf.Logger,
		collectors:  conf.Collectors,
	}
}

func (g *NodeGroup) Add(maxWorkers int, maxQueueSize int) {
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
	var slogger *simpleLogger
	var cLogger *log.Logger
	var gLogger *log.Logger

	if logTel != nil {
		slogger = NewSimpleLogger(logTel.GeneralLog(net.PeerID()), Debug)
		cLogger = log.New(logTel.ContractLog(net.PeerID()), "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		gLogger = log.New(logTel.GeneralLog(net.PeerID()), "[general] ", log.Ldate|log.Ltime|log.Lmicroseconds)
	} else {
		slogger = NewSimpleLogger(io.Discard, Critical)
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
		g.encoder,
		maxWorkers,
		maxQueueSize,
		pluginconfig.DefaultCacheExpiration,
		pluginconfig.DefaultCacheClearInterval,
	)

	coordFac := &coordinator.CoordinatorFactory{
		Logger:     gLogger,
		Encoder:    g.encoder,
		Logs:       ct,
		CacheClean: time.Minute,
	}

	cObsFac := &polling.PollingObserverFactory{
		Logger:  gLogger,
		Source:  ct,
		Heads:   ct,
		Runner:  runr,
		Encoder: g.encoder,
	}

	dConfig := ocr2keepers.DelegateConfig{
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
		ConditionalObserverFactory: cObsFac,
		CoordinatorFactory:         coordFac,
		Encoder:                    g.encoder,
		Runner:                     runr,
		Logger:                     slogger,
		MonitoringEndpoint:         g.monitor,
		OffchainConfigDigester:     g.digester,
		OffchainKeyring:            offchainRing,
		OnchainKeyring:             onchainRing,
		MaxServiceWorkers:          maxWorkers,
		ServiceQueueLength:         maxQueueSize,
	}

	service, err := ocr2keepers.NewDelegate(dConfig)
	if err != nil {
		panic(err)
	}

	g.nodes[net.PeerID()] = &ActiveNode{
		Service:  service,
		Contract: ct,
	}

	g.logger.Println("starting new node")
	_ = service.Start(context.Background())
	ct.Start()

	g.confLoader.AddSigner(net.PeerID(), onchainRing, offchainRing)
}

func (g *NodeGroup) Start(ctx context.Context, count int, maxWorkers int, maxQueueSize int) error {
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

func (g *NodeGroup) ReportResults() {
	var keyIDLookup map[string][]string
	for _, col := range g.collectors {
		switch ct := col.(type) {
		case *telemetry.RPCCollector:
			err := ct.WriteResults()
			if err != nil {
				panic(err)
			}
		case *telemetry.ContractEventCollector:
			_, keyIDLookup = ct.Data()
		}
	}

	ub, err := newUpkeepStatsBuilder(g.upkeeps, g.transmitter.Results(), keyIDLookup, g.encoder)
	if err != nil {
		g.logger.Printf("stats builder failure: %s", err)
	}

	g.logger.Println("================ summary ================")

	totalIDChecks := 0
	totalIDs := 0
	totalEligibles := 0
	totalPerforms := 0
	totalMisses := 0
	var avgPerformDelay float64 = -1
	var avgCheckDelay float64 = -1
	idCheckData := []int{}

	for _, id := range ub.UpkeepIDs() {
		stats := ub.UpkeepStats(id)
		totalIDs++
		totalEligibles += stats.Eligible
		totalMisses += stats.Missed
		totalPerforms += stats.Eligible - stats.Missed

		if stats.AvgPerformDelay >= 0 {
			if avgPerformDelay < 0 {
				avgPerformDelay = stats.AvgPerformDelay
			} else {
				avgPerformDelay = (avgPerformDelay + stats.AvgPerformDelay) / 2
			}
		}

		if stats.AvgCheckDelay >= 0 {
			if avgCheckDelay < 0 {
				avgCheckDelay = stats.AvgCheckDelay
			} else {
				avgCheckDelay = (avgCheckDelay + stats.AvgCheckDelay) / 2
			}
		}

		checks, checked := keyIDLookup[id]
		if checked {
			totalIDChecks += len(checks)
			idCheckData = append(idCheckData, len(checks))
		} else {
			idCheckData = append(idCheckData, 0)
		}

		if stats.Missed != 0 {
			g.logger.Printf("%s was missed %d times", id, stats.Missed)
			g.logger.Printf("%s was eligible at %s", id, strings.Join(ub.Eligibles(id), ", "))

			by := []string{}
			for _, tr := range ub.TransmitEvents(id) {
				v := fmt.Sprintf("[address=%s, epoch=%d, round=%d, block=%s]", tr.SendingAddress, tr.Epoch, tr.Round, tr.InBlock)
				by = append(by, v)
			}
			g.logger.Printf("%s transactions %s", id, strings.Join(by, ", "))

			if checked {
				g.logger.Printf("%s was checked at %s", id, strings.Join(checks, ", "))
			}
		}
	}

	g.logger.Printf("total ids: %d", totalIDs)
	g.logger.Printf("total checks by network: %d", totalIDChecks)

	g.logger.Printf(" ---- Statistics / Checks per ID ---")

	sort.Slice(idCheckData, func(i, j int) bool {
		return idCheckData[i] < idCheckData[j]
	})

	// average
	avgChecksPerID := float64(totalIDChecks) / float64(len(idCheckData))
	g.logger.Printf("average: %0.2f", avgChecksPerID)

	// median
	median, q1Data, q3Data := findMedianAndSplitData(idCheckData)
	q1, lowerOutliers, _ := findMedianAndSplitData(q1Data)
	q3, _, upperOutliers := findMedianAndSplitData(q3Data)
	iqr := q3 - q1

	g.logger.Printf("IQR: %0.2f", iqr)
	inIQR := 0
	for i := 0; i < len(idCheckData); i++ {
		if float64(idCheckData[i]) >= q1 && float64(idCheckData[i]) <= q3 {
			inIQR++
		}
	}
	g.logger.Printf("IQR percent of whole: %0.2f%s", float64(inIQR)/float64(len(idCheckData))*100, "%")

	lowest, lOutliers := findLowestAndOutliers(q1-(1.5*iqr), lowerOutliers)
	if lOutliers > 0 {
		g.logger.Printf("lowest value: %d", lowest)
		g.logger.Printf("lower outliers (count): %d", lOutliers)
	} else {
		g.logger.Printf("no outliers below lower fence")
	}

	g.logger.Printf("Lower Fence (Q1 - 1.5*IQR): %0.2f", q1-(1.5*iqr))
	g.logger.Printf("Q1: %0.2f", q1)
	g.logger.Printf("Median: %0.2f", median)
	g.logger.Printf("Q3: %0.2f", q3)
	g.logger.Printf("Upper Fence (Q3 + 1.5*IQR): %0.2f", q3+(1.5*iqr))

	highest, hOutliers := findHighestAndOutliers(q3+(1.5*iqr), upperOutliers)
	if hOutliers > 0 {
		g.logger.Printf("highest value: %d", highest)
		g.logger.Printf("upper outliers (count): %d", hOutliers)
	} else {
		g.logger.Printf("no outliers above upper fence")
	}

	g.logger.Printf(" ---- end ---")

	g.logger.Printf(" ---- Statistics / Transmits per Node (account) ---")
	accStats := ub.Transmits()
	for _, acc := range accStats {
		g.logger.Printf("account %s transmitted %d times (%.2f%s)", acc.Account, acc.Count, acc.Pct, "%")
	}
	g.logger.Printf(" ---- end ---")

	// average perform delay
	g.logger.Printf("average perform delay: %d blocks", int(math.Round(avgPerformDelay)))
	g.logger.Printf("average check delay: %d blocks", int(math.Round(avgCheckDelay)))
	g.logger.Printf("total eligibles: %d", totalEligibles)
	g.logger.Printf("total performs in a transaction: %d", totalPerforms)
	g.logger.Printf("total confirmed misses: %d", totalMisses)
	g.logger.Println("================ end ================")
}
