package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	flag "github.com/spf13/pflag"
)

var (
	contract  = flag.StringP("contract", "c", "", "contract address")
	rpcNode   = flag.StringP("rpc-node", "n", "https://rinkeby.infura.io", "rpc host to make calls")
	out       = flag.StringP("print-reports", "p", "", "directory to print reports to. prints one report per file")
	nodes     = flag.IntP("nodes", "s", 3, "number of nodes to simulate")
	rounds    = flag.IntP("rounds", "d", 2, "OCR rounds to simulate; 0 for no limit")
	rndTime   = flag.IntP("round-time", "t", 5000, "round time in milliseconds")
	qTime     = flag.IntP("query-time", "q", 0, "max time to run a Query operation in milliseconds")
	oTime     = flag.IntP("observation-time", "o", 0, "max time to run an Observation operation in milliseconds")
	rTime     = flag.IntP("report-time", "r", 0, "max time to run a Report operation in milliseconds")
	maxRun    = flag.IntP("max-run-time", "m", 0, "max run time in seconds for the simulation")
	profiler  = flag.Bool("pprof", false, "run pprof server on startup")
	pprofPort = flag.Int("pprof-port", 6060, "port to serve the profiler on")
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

	log.Println("starting simulation")

	config := &SimulatorConfig{
		ContractAddress:      contract,
		RPC:                  rpcNode,
		Nodes:                nodes,
		Rounds:               rounds,
		RoundTime:            rndTime,
		QueryTimeLimit:       qTime,
		ObservationTimeLimit: oTime,
		ReportTimeLimit:      rTime,
		MaxRunTime:           maxRun,
	}

	if *out != "" {
		config.ReportOutputPath = out
	}

	err := runFullSimulation(log.Default(), config)
	if err != nil {
		panic(err)
	}
}

type logWriter struct {
	l commontypes.Logger
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.l.Debug(string(p), nil)
	n = len(p)
	return
}
