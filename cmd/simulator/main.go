package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

var (
	contract = flag.String("contract", "", "contract address")
	rpc      = flag.String("rpc", "https://rinkeby.infura.io", "rpc host to make calls")
	nodes    = flag.Int("nodes", 3, "number of nodes to simulate")
	rounds   = flag.Int("rounds", 2, "OCR rounds to simulate; 0 for no limit")
	rtime    = flag.Int("round-time", 5, "round time in seconds")
)

func main() {
	flag.Parse()

	if contract == nil || *contract == "" {
		panic("contract must be defined")
	}

	if rpc == nil || *rpc == "" {
		panic("rpc must be defined")
	}

	address := common.HexToAddress(*contract)
	receivers := make([]*OCRReceiver, *nodes)
	for i := 0; i < *nodes; i++ {
		receivers[i] = NewOCRReceiver(fmt.Sprintf("node %d", i+1))
	}

	t := time.Duration(int64(*rtime))
	controller := NewOCRController(t*time.Second, *rounds, receivers...)

	for _, rec := range receivers {
		wrapPluginReceiver(controller, rec, makePlugin(address))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-controller.Start(ctx):
		log.Printf("done signal encountered")
		cancel()
	case <-ctx.Done():
		cancel()
	case <-c:
		cancel()
	}
}

func makePlugin(address common.Address) types.ReportingPlugin {
	client, err := ethclient.Dial(*rpc)
	if err != nil {
		panic(err)
	}

	reg, err := chain.NewEVMRegistryV1_2(address, client)
	if err != nil {
		panic(err)
	}

	factory := keepers.NewReportingPluginFactory(reg, chain.NewEVMReportEncoder())
	plugin, _, err := factory.NewReportingPlugin(types.ReportingPluginConfig{})
	if err != nil {
		panic(err)
	}

	return plugin
}

func wrapPluginReceiver(controller *OCRController, receiver *OCRReceiver, plugin types.ReportingPlugin) {
	go func(c *OCRController, r *OCRReceiver, p types.ReportingPlugin) {
		for {
			select {
			case call := <-receiver.Init:
				log.Printf("%s: init call received", receiver.Name)
				q, err := p.Query(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)})
				if err != nil {
					log.Printf("fatal error in query: %s", err)
					return
				}
				go func() {
					select {
					case c.Queries <- OCRQuery(q):
						log.Printf("%s: sent query to controller", receiver.Name)
						return
					case <-call.Context.Done():
						return
					}
				}()
			case call := <-receiver.Query:
				log.Printf("%s: query recieved", receiver.Name)
				o, err := p.Observation(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Query(call.Data))
				if err != nil {
					log.Printf("fatal error in query: %s", err)
				}
				go func() {
					select {
					case c.Observations <- OCRObservation(o):
						log.Printf("%s: sent observation to controller", receiver.Name)
						return
					case <-call.Context.Done():
						return
					}
				}()
			case call := <-receiver.Observations:
				log.Printf("%s: observations received", receiver.Name)
				attr := make([]types.AttributedObservation, len(call.Data))
				for i, o := range call.Data {
					attr[i] = types.AttributedObservation{
						Observation: types.Observation(o),
					}
				}

				b, r, err := p.Report(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Query{}, attr)
				if err != nil {
					log.Printf("fatal error in query: %s", err)
				}

				go func() {
					rv := types.Report{}
					if b {
						rv = r
					}
					select {
					case c.Reports <- OCRReport(rv):
						log.Printf("%s: sent report to controller", receiver.Name)
						return
					case <-call.Context.Done():
						return
					}
				}()
			case call := <-receiver.Report:
				log.Printf("%s: report received", receiver.Name)
				b, err := p.ShouldAcceptFinalizedReport(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Report(call.Data))
				if err != nil {
					log.Printf("fatal error in query: %s", err)
				}

				log.Printf("accept finalized report for round: %d; epoch: %d: %t", call.Round, call.Epoch, b)
			case <-r.Stop:
				return
			}
		}
	}(controller, receiver, plugin)
}
