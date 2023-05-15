package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	flag "github.com/spf13/pflag"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/simulators"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/encoding"
)

var (
	simulationFile  = flag.StringP("simulation-file", "f", "runbook.json", "file path to read simulation config from")
	outputDirectory = flag.StringP("output-directory", "o", "./runbook_logs", "directory path to output log files")
	simulate        = flag.Bool("simulate", false, "run simulation")
	displayCharts   = flag.Bool("charts", false, "create and serve charts")
	profiler        = flag.Bool("pprof", false, "run pprof server on startup")
	pprofPort       = flag.Int("pprof-port", 6060, "port to serve the profiler on")
)

func main() {
	flag.Parse()

	if *profiler {
		log.Println("starting profiler; waiting 5 seconds to start simulation")
		go func() {
			log.Println(http.ListenAndServe(fmt.Sprintf("localhost:%d", *pprofPort), nil))
		}()
		<-time.After(5 * time.Second)
	}

	err := os.MkdirAll(*outputDirectory, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	dat, err := os.ReadFile(*simulationFile)
	if err != nil {
		panic(err)
	}

	var rb config.RunBook
	err = json.Unmarshal(dat, &rb)
	if err != nil {
		panic(err)
	}

	if *simulate {
		f, err := os.OpenFile(fmt.Sprintf("%s/simulation.log", *outputDirectory), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)

		js, err := os.OpenFile(fmt.Sprintf("%s/runbook.json", *outputDirectory), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer js.Close()

		_, err = js.Write(dat)
		if err != nil {
			log.Fatalf("error writing to runbook file: %s", err)
		}
	}

	log.Println("generating upkeeps")
	upkeeps, err := simulators.GenerateSimulatedUpkeeps(rb)
	if err != nil {
		panic(err)
	}

	// generic report encoder for testing evm encoding/decoding
	enc := FullEncoder{}

	// generic config digester
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1,
		ContractAddress: common.BigToAddress(big.NewInt(12)),
	}

	rpcC := telemetry.NewNodeRPCCollector(*outputDirectory)
	logC := telemetry.NewNodeLogCollector(*outputDirectory)
	ctrC := telemetry.NewContractEventCollector(*outputDirectory, enc)

	ngConf := NodeGroupConfig{
		Digester:      digester,
		Cadence:       rb.BlockCadence,
		Encoder:       enc,
		Upkeeps:       upkeeps,
		ConfigEvents:  rb.ConfigEvents,
		MonitorIO:     io.Discard, // TODO: the monitor data format is not text. not sure what to make of it yet.
		RPCConfig:     rb.RPCDetail,
		AvgNetLatency: rb.AvgNetworkLatency.Value(),
		LogPath:       *outputDirectory,
		Logger:        log.Default(),
		Collectors:    []telemetry.Collector{rpcC, logC, ctrC},
	}
	ng := NewNodeGroup(ngConf)

	var wg sync.WaitGroup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	if *simulate {
		wg.Add(1)
		go func(ct context.Context, b config.RunBook) {
			if err := ng.Start(ct, b.Nodes, b.MaxServiceWorkers, b.MaxQueueSize); err != nil {
				log.Printf("%s", err)
			}

			wg.Done()
		}(ctx, rb)
	}

	var server *http.Server
	if *displayCharts {
		wg.Add(1)
		go func() {
			http.HandleFunc("/", rpcC.SummaryChart())
			ln, _ := net.Listen("tcp", "localhost:8081")
			server = &http.Server{}
			_ = server.Serve(ln)
			wg.Done()
		}()
	}

	<-c

	if server != nil {
		_ = server.Shutdown(ctx)
	}

	cancel()
	wg.Wait()
}

type FullEncoder struct {
	simulators.SimulatedReportEncoder
	encoding.KeyBuilder
}
