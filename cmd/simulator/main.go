package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	flag "github.com/spf13/pflag"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/node"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/run"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/telemetry"
)

var (
	simulationFile  = flag.StringP("simulation-file", "f", "./simulation_plan.json", "file path to read simulation config from")
	outputDirectory = flag.StringP("output-directory", "o", "./simulation_plan_logs", "directory path to output log files")
	simulate        = flag.Bool("simulate", false, "run simulation")
	serveCharts     = flag.Bool("charts", false, "create and serve charts")
	profiler        = flag.Bool("pprof", false, "run pprof server on startup")
	pprofPort       = flag.Int("pprof-port", 6060, "port to serve the profiler on")
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
	plan, err := run.LoadSimulationPlan(*simulationFile)
	if err != nil {
		procLog.Printf("failed to initialize simulation plan: %s", err)
		os.Exit(1)
	}

	// ----- setup simulation output directory and file handles
	outputs, err := run.SetupOutput(*outputDirectory, *simulate, plan)
	if err != nil {
		procLog.Printf("failed to setup output directory: %s", err)
		os.Exit(1)
	}

	// ----- create simulated upkeeps from simulation plan
	upkeeps, err := chain.GenerateAllUpkeeps(plan)
	if err != nil {
		procLog.Printf("failed to generate simulated upkeeps: %s", err)
		os.Exit(1)
	}

	ngConf := node.GroupConfig{
		SimulationPlan: plan,
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
		Logger: outputs.SimulationLog,
	}

	fmt.Printf("Starting simulation ...\n\n")

	progress := telemetry.NewProgressTelemetry(os.Stdout)
	progress.Start()

	ng, err := node.NewGroup(ngConf, progress)
	if err != nil {
		procLog.Printf("failed to create node group: %s", err)
		os.Exit(1)
	}

	ctx, cancel := contextWithInterrupt(context.Background())

	var wg sync.WaitGroup
	if *simulate {
		wg.Add(1)
		go func(serviceCtx context.Context, simPlan config.SimulationPlan, logger *log.Logger) {
			if err := ng.Start(serviceCtx, simPlan.Node); err != nil {
				logger.Printf("node group closed with error: %s", err)
			}

			if err := progress.Close(); err != nil {
				logger.Printf("failed to close progress tracker: %s", err)
			}

			wg.Done()
		}(ctx, plan, procLog)
	}

	if *serveCharts {
		var server *http.Server

		// attempt to start the chart server
		wg.Add(1)
		go func(logger *log.Logger) {
			http.HandleFunc("/", outputs.RPCCollector.SummaryChart())

			listener, err := net.Listen("tcp", "localhost:8081")
			if err != nil {
				logger.Printf("failed to start chart server: %s", err)

				// cancel the service context to close all services
				cancel()
			}

			server = &http.Server{}

			if err := server.Serve(listener); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					// set the server to nil to ensure the shutdown method
					// does not get applied
					server = nil

					logger.Printf("chart server failure: %s", err)

					// cancel the service context to close all services
					cancel()
				}
			}

			wg.Done()
		}(procLog)

		// listen for context closure to stop the chart server
		wg.Add(1)
		go func(serviceCtx context.Context) {
			<-serviceCtx.Done()

			if server != nil {
				shutdownCtx, shutdownCancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Second))

				_ = server.Shutdown(shutdownCtx)

				shutdownCancel()
			}

			wg.Done()
		}(ctx)
	}

	wg.Wait()

	fmt.Printf("\nSimulation done")

	if !progress.AllProgressComplete() {
		fmt.Println("\nsimulation failed")
		os.Exit(1)
	}
}
