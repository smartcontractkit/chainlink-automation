package main

import (
	"context"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	flag "github.com/spf13/pflag"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/node"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/run"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/upkeep"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
)

var (
	simulationFile  = flag.StringP("simulation-file", "f", "./runbook.json", "file path to read simulation config from")
	outputDirectory = flag.StringP("output-directory", "o", "./runbook_logs", "directory path to output log files")
	simulate        = flag.Bool("simulate", false, "run simulation")
	//displayCharts   = flag.Bool("charts", false, "create and serve charts")
	profiler  = flag.Bool("pprof", false, "run pprof server on startup")
	pprofPort = flag.Int("pprof-port", 6060, "port to serve the profiler on")
)

func main() {
	// ----- collect run parameters
	flag.Parse()

	procLog := log.New(log.Writer(), "[simulator-startup] ", log.LstdFlags)

	// ----- start run profiler if configured
	run.Profiler(run.ProfilerConfig{
		Enabled:   *profiler,
		PprofPort: *pprofPort,
		Wait:      5 * time.Second,
	}, procLog)

	// ----- read simulation file
	procLog.Println("loading simulation assets ...")
	rb, err := run.LoadRunBook(*simulationFile)
	if err != nil {
		procLog.Printf("failed to initialize runbook: %s", err)
		os.Exit(1)
	}

	// ----- setup simulation output directory and file handles
	outputs, err := run.SetupOutput(*outputDirectory, *simulate, rb)
	if err != nil {
		procLog.Printf("failed to setup output directory: %s", err)
		os.Exit(1)
	}

	// ----- create simulated upkeeps from runbook
	procLog.Println("generating simulated upkeeps ...")
	upkeeps, err := upkeep.GenerateConditionals(rb)
	if err != nil {
		procLog.Printf("failed to generate simulated upkeeps: %s", err)
		os.Exit(1)
	}

	ngConf := node.GroupConfig{
		Runbook: rb,
		// Digester is a generic offchain digester
		Digester: evmutil.EVMOffchainConfigDigester{
			ChainID:         1,
			ContractAddress: common.BigToAddress(big.NewInt(12)),
		},
		Upkeeps: upkeeps,
		Collectors: []telemetry.Collector{
			outputs.RPCCollector,
			outputs.LogCollector,
			outputs.EventCollector,
		},
		Logger: procLog,
	}
	ng := node.NewGroup(ngConf)

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
	/*
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
	*/

	<-c

	if server != nil {
		_ = server.Shutdown(ctx)
	}

	cancel()
	wg.Wait()
}
