package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func runFullSimulation(contract, rpc, out string, nodes, rounds, rndTime, qTime, oTime, rTime, maxRun int) error {
	logger := NewSimpleLogger(os.Stdout, Debug)
	defer func() {
		if err := recover(); err != nil {
			logger.Critical(fmt.Sprint(err), nil)
			debug.PrintStack()
		}
	}()

	w := &logWriter{l: logger}
	log.SetOutput(w)
	log.SetPrefix("[simulator] ")
	log.SetFlags(log.Lshortfile | log.Lmsgprefix)

	if contract == "" {
		return fmt.Errorf("contract must be defined")
	}

	if rpc == "" {
		return fmt.Errorf("rpc must be defined")
	}

	if nodes <= 0 {
		return fmt.Errorf("number of nodes must be greater than 0")
	}

	if rounds < 0 {
		return fmt.Errorf("number of rounds must be greater than or equal to 0")
	}

	if rndTime < 0 {
		return fmt.Errorf("round time must be greater than or equal to 0")
	}

	if maxRun < 0 {
		return fmt.Errorf("max run time must be greater than or equal to 0")
	}

	address := common.HexToAddress(contract)
	receivers := make([]*OCRReceiver, nodes)
	for i := 0; i < nodes; i++ {
		receivers[i] = NewOCRReceiver(fmt.Sprintf("node %d", i+1))
	}

	t := time.Duration(int64(rndTime))
	lg := log.New(w, "[controller] ", log.Lshortfile|log.Lmsgprefix)
	controller := NewOCRController(t*time.Second, rounds, lg, receivers...)
	controller.QueryTime = time.Duration(int64(qTime)) * time.Second
	controller.ObservationTime = time.Duration(int64(oTime)) * time.Second
	controller.ReportTime = time.Duration(int64(rTime)) * time.Second

	for i, rec := range receivers {
		l := log.New(w, fmt.Sprintf("[node %d] ", i+1), log.Lshortfile|log.Lmsgprefix)
		wrapPluginReceiver(controller, rec, makePlugin(address, controller, l))
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if maxRun > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int64(maxRun))*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Print("starting simulation")
	select {
	case <-controller.Start(ctx):
		log.Print("done signal encountered")
		cancel()
	case <-ctx.Done():
		cancel()
	case <-c:
		cancel()
	}

	if out != "" {
		log.Print("writing reports")
		controller.WriteReports(out)
	}

	return nil
}

func makePlugin(address common.Address, controller *OCRController, logger *log.Logger) types.ReportingPlugin {
	client, err := ethclient.Dial(*rpc)
	if err != nil {
		panic(err)
	}

	reg, err := chain.NewEVMRegistryV1_2(address, client)
	if err != nil {
		panic(err)
	}

	factory := keepers.NewReportingPluginFactory(reg, chain.NewEVMReportEncoder(), logger)
	plugin, info, err := factory.NewReportingPlugin(types.ReportingPluginConfig{})
	if err != nil {
		panic(err)
	}

	controller.MaxQueryLength = info.Limits.MaxQueryLength
	controller.MaxObservationLength = info.Limits.MaxObservationLength
	controller.MaxReportLength = info.Limits.MaxReportLength

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
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}
				go func() {
					select {
					case c.Queries <- OCRQuery(q):
						log.Printf("%s: sent query to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.QueryTime > 0 {
							log.Printf("%s: context ended for query call", receiver.Name)
						}
						return
					}
				}()
			case call := <-receiver.Query:
				log.Printf("%s: query recieved", receiver.Name)
				o, err := p.Observation(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Query(call.Data))
				if err != nil {
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}
				go func() {
					select {
					case c.Observations <- OCRObservation(o):
						log.Printf("%s: sent observation to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.ObservationTime > 0 {
							log.Printf("%s: context ended for observation call", receiver.Name)
						}
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
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}

				go func() {
					rv := r
					if !b {
						log.Printf("%s: nothing to report; sending empty report", receiver.Name)
						rv = types.Report{}
					}
					select {
					case c.Reports <- OCRReport(rv):
						log.Printf("%s: sent report to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.ReportTime > 0 {
							log.Printf("%s: context ended for report call", receiver.Name)
						}
						return
					}
				}()
			case call := <-receiver.Report:
				log.Printf("%s: report received", receiver.Name)
				b, err := p.ShouldAcceptFinalizedReport(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Report(call.Data))
				if err != nil {
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}

				log.Printf("accept finalized report for round: %d; epoch: %d: %t", call.Round, call.Epoch, b)
			case <-r.Stop:
				return
			}
		}
	}(controller, receiver, plugin)
}
