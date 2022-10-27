package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/blocks"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/simulators"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
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
	Encoder       ktypes.ReportEncoder
	Upkeeps       []simulators.SimulatedUpkeep
	ConfigEvents  []config.ConfigEvent
	RPCConfig     config.RPC
	AvgNetLatency time.Duration
	MonitorIO     io.Writer
	LogPath       string
	Logger        *log.Logger
}

type NodeGroup struct {
	nodes       map[string]*ActiveNode
	network     *simulators.SimulatedNetwork
	digester    types.OffchainConfigDigester
	blockSrc    *blocks.BlockBroadcaster
	encoder     ktypes.ReportEncoder
	transmitter *blocks.TransmitLoader
	confLoader  *blocks.ConfigLoader
	upkeeps     []simulators.SimulatedUpkeep
	monitor     commontypes.MonitoringEndpoint
	rpcConf     config.RPC
	logPath     string
	logger      *log.Logger
	openFiles   map[string][]*os.File
	telemetry   *ResultsTelemetry
}

func NewNodeGroup(conf NodeGroupConfig) *NodeGroup {
	transmit := blocks.NewTransmitLoader()
	confLoad := blocks.NewConfigLoader(conf.ConfigEvents, conf.Digester)

	// general monitoring collector
	monitor := NewMonitorToWriter(conf.MonitorIO)

	return &NodeGroup{
		nodes:       make(map[string]*ActiveNode),
		openFiles:   make(map[string][]*os.File),
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
		telemetry:   NewResultsTelemetry(),
	}
}

func (g *NodeGroup) Add() {
	net := g.network.NewFactory()

	offchainRing, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)
	if err != nil {
		panic(err)
	}

	onchainRing, err := config.NewEVMKeyring(rand.Reader)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/oracle_%s.log", g.logPath, net.PeerID()), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		g.logger.Println(err.Error())
		f.Close()
		return
	}

	cLog, err := os.OpenFile(fmt.Sprintf("%s/oracle_%s_contract.log", g.logPath, net.PeerID()), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		g.logger.Println(err.Error())
		cLog.Close()
		return
	}

	// general logger
	slogger := NewSimpleLogger(f, Debug)
	cLogger := log.New(cLog, "[contract] ", log.Ldate|log.Ltime|log.Lmicroseconds)

	ct := simulators.NewSimulatedContract(g.blockSrc, g.digester, g.upkeeps, g.encoder, g.transmitter, g.rpcConf.AverageLatency, onchainRing.PKString(), g.telemetry, cLogger)
	db := simulators.NewSimulatedDatabase()

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
		Logger:                 slogger,
		MonitoringEndpoint:     g.monitor,
		OffchainConfigDigester: g.digester,
		OffchainKeyring:        offchainRing,
		OnchainKeyring:         onchainRing,
		Registry:               ct,
		PerformLogProvider:     ct,
		ReportEncoder:          g.encoder,
		MaxServiceWorkers:      10,
		ServiceQueueLength:     100,
	}

	service, err := ocr2keepers.NewDelegate(dConfig)
	if err != nil {
		panic(err)
	}

	g.openFiles[net.PeerID()] = []*os.File{f, cLog}
	g.nodes[net.PeerID()] = &ActiveNode{
		Service:  service,
		Contract: ct,
	}

	g.logger.Println("starting new node")
	_ = service.Start(context.Background())
	ct.Start()

	g.confLoader.AddSigner(net.PeerID(), onchainRing, offchainRing)
}

func (g *NodeGroup) Start(count int) error {
	var err error

	for i := 0; i < count; i++ {
		g.Add()
	}

	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Print("starting simulation")
	select {
	case <-g.blockSrc.Start():
		err = fmt.Errorf("block duration ended")
	case <-c:
		g.blockSrc.Stop()
		err = fmt.Errorf("SIGTERM event stopping process")
	}

	g.ReportResults()

	for id, node := range g.nodes {
		if err := node.Service.Close(); err != nil {
			log.Printf("error encountered while attempting to close down node '%s': %s", id, err)
		}
		node.Contract.Stop()
		files, ok := g.openFiles[id]
		if ok {
			for _, f := range files {
				f.Close()
			}
		}
	}

	return err
}

func (g *NodeGroup) ReportResults() {

	upkeepLookup := make(map[string]simulators.SimulatedUpkeep)
	performsById := make(map[string][]string)
	var uCount int
	for _, up := range g.upkeeps {
		upkeepLookup[up.ID.String()] = up
		performsById[up.ID.String()] = []string{}
		uCount += len(up.EligibleAt)
	}

	// performed upkeeps
	// missed upkeeps
	var avgPerformDelay float64
	trAccounts := make(map[string]int)

	// report results
	transmits := g.transmitter.Results()
	var performsInTransactions int
	for _, tr := range transmits {
		block, ok := new(big.Int).SetString(tr.InBlock, 10)
		if !ok {
			g.logger.Printf("could not parse block: %s", tr.InBlock)
			continue
		}

		_, ok = trAccounts[tr.SendingAddress]
		if !ok {
			trAccounts[tr.SendingAddress] = 0
		}
		trAccounts[tr.SendingAddress]++

		trResults, err := g.encoder.DecodeReport(tr.Report)
		if err != nil {
			g.logger.Printf("error decoding report: %s", err)
			continue
		}

		for _, trResult := range trResults {
			delay := float64(new(big.Int).Sub(block, big.NewInt(int64(trResult.CheckBlockNumber))).Int64())

			if avgPerformDelay == 0 {
				avgPerformDelay = delay
			} else {
				avgPerformDelay = (avgPerformDelay + delay) / 2
			}
			performsInTransactions++

			parts := strings.Split(string(trResult.Key), "|")
			_, ok := performsById[parts[1]]
			if ok {
				performsById[parts[1]] = append(performsById[parts[1]], block.String())
			}
		}
	}

	g.logger.Println("================ results ================")

	var missed int
	for upID, upkeep := range upkeepLookup {
		eligibles := len(upkeep.EligibleAt)
		var performs int

		performStrings, ok := performsById[upID]
		if ok {
			performs = len(performStrings)
		}

		missed += eligibles - performs
		if eligibles-performs > 0 {
			g.logger.Printf("%s was missed %d times", upID, eligibles-performs)

			printEligibles := make([]string, len(upkeep.EligibleAt))
			for i, el := range upkeep.EligibleAt {
				printEligibles[i] = el.String()
			}

			g.logger.Printf("%s was eligible at %s", upID, strings.Join(printEligibles, ", "))
			g.logger.Printf("%s was performed at %s", upID, strings.Join(performStrings, ", "))

			keys, ok := g.telemetry.keyIDLookup[upID]
			if ok {
				g.logger.Printf("%s was checked at %s", upID, strings.Join(keys, ", "))
			}
		}
	}

	perc := make(map[string]float64)
	// transmit account distribution
	for acc, trs := range trAccounts {
		perc[acc] = float64(trs) / float64(len(transmits)) * 100
	}

	for acc, p := range perc {
		g.logger.Printf("account %s transmitted %d times (%.2f percent)", acc, trAccounts[acc], p)
	}

	// average perform delay
	g.logger.Printf("average perform delay: %d blocks", int(math.Round(avgPerformDelay)))
	g.logger.Printf("total eligibles: %d", uCount)
	g.logger.Printf("total performs in a transaction: %d", performsInTransactions)
	g.logger.Printf("total confirmed misses: %d", missed)
	g.logger.Println("================ end ================")
}

type ResultsTelemetry struct {
	mu          sync.Mutex
	keyChecks   map[string]int
	keyIDLookup map[string][]string
}

func NewResultsTelemetry() *ResultsTelemetry {
	return &ResultsTelemetry{
		keyChecks:   make(map[string]int),
		keyIDLookup: make(map[string][]string),
	}
}

func (rt *ResultsTelemetry) CheckKey(key []byte) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	k := string(key)
	parts := strings.Split(k, "|")

	_, ok := rt.keyChecks[k]
	if !ok {
		rt.keyChecks[k] = 0
	}
	rt.keyChecks[k]++

	val, ok := rt.keyIDLookup[parts[1]]
	if !ok {
		rt.keyIDLookup[parts[1]] = []string{parts[0]}
	} else {
		var found bool
		for _, v := range val {
			if v == parts[0] {
				found = true
			}
		}

		if !found {
			rt.keyIDLookup[parts[1]] = append(val, parts[0])
		}
	}
}
