package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	flag "github.com/spf13/pflag"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/simulators"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

var (
	simulationFile  = flag.StringP("simulation-file", "f", "runbook.json", "file path to read simulation config from")
	outputDirectory = flag.StringP("output-directory", "o", "./runbook_logs", "directory path to output log files")
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

	f, err := os.OpenFile(fmt.Sprintf("%s/simulation.log", *outputDirectory), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	dat, err := os.ReadFile(*simulationFile)
	if err != nil {
		panic(err)
	}

	var rb config.RunBook
	err = json.Unmarshal(dat, &rb)
	if err != nil {
		panic(err)
	}

	log.Println("generating upkeeps")
	upkeeps, err := simulators.GenerateSimulatedUpkeeps(rb)
	if err != nil {
		panic(err)
	}

	// generic report encoder for testing evm encoding/decoding
	enc := chain.NewEVMReportEncoder()

	// generic config digester
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1,
		ContractAddress: common.BigToAddress(big.NewInt(12)),
	}

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
	}
	ng := NewNodeGroup(ngConf)

	if err := ng.Start(rb.Nodes, rb.MaxServiceWorkers, rb.MaxQueueSize); err != nil {
		log.Printf("%s", err)
	}
}
