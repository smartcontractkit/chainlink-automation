package main

import (
	"github.com/smartcontractkit/libocr/commontypes"
	flag "github.com/spf13/pflag"
)

var (
	contract = flag.StringP("contract", "c", "", "contract address")
	rpc      = flag.StringP("rpc-node", "n", "https://rinkeby.infura.io", "rpc host to make calls")
	out      = flag.StringP("print-reports", "p", "", "directory to print reports to. prints one report per file")
	nodes    = flag.IntP("nodes", "s", 3, "number of nodes to simulate")
	rounds   = flag.IntP("rounds", "d", 2, "OCR rounds to simulate; 0 for no limit")
	rndTime  = flag.IntP("round-time", "t", 5, "round time in seconds")
	qTime    = flag.IntP("query-time", "q", 0, "max time to run a Query operation in seconds")
	oTime    = flag.IntP("observation-time", "o", 0, "max time to run an Observation operation in seconds")
	rTime    = flag.IntP("report-time", "r", 0, "max time to run a Report operation in seconds")
	maxRun   = flag.IntP("max-run-time", "m", 0, "max run time in seconds for the simulation")
)

func main() {
	flag.Parse()

	err := runFullSimulation(*contract, *rpc, *out, *nodes, *rounds, *rndTime, *qTime, *oTime, *rTime, *maxRun)
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
