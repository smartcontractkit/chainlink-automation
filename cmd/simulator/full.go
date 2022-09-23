package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

var (
	ErrInvalidConfig = fmt.Errorf("invalid simulator configuration")
)

// SimulatorConfig ...
type SimulatorConfig struct {
	// ContractAddress is the registry contract where the simulator checks upkeeps
	ContractAddress *string
	// RPC is the url where ethereum calls are sent
	RPC *string
	// ReportOutputPath is the file path where reports will be written to for each round
	ReportOutputPath *string
	// Nodes is the total number of nodes to simulate
	Nodes *int
	// Rounds is the total number of rounds to simulate; 0 or nil is no limit
	Rounds *int
	// RoundTime limits the time a round can take in milliseconds; 0 or nil is no limit
	RoundTime *int
	// QueryTimeLimit limits the time a query can take in milliseconds; 0 or nil is no limit
	QueryTimeLimit *int
	// ObservationTimeLimit limits the time an observation can take in milliseconds; 0 or nil is no limit
	ObservationTimeLimit *int
	// ReportTimeLimit limits the time a report can take in milliseconds; 0 or nil is no limit
	ReportTimeLimit *int
	// MaxRunTime limits the time a simulation runs in seconds; 0 or nil is no limit
	MaxRunTime *int
}

func validateSimulatorConfig(config *SimulatorConfig) error {
	if config == nil {
		return fmt.Errorf("%w: nil config", ErrInvalidConfig)
	}

	if config.ContractAddress == nil {
		return fmt.Errorf("%w: contract address cannot be nil", ErrInvalidConfig)
	}

	if _, err := common.NewMixedcaseAddressFromString(*config.ContractAddress); err != nil {
		return fmt.Errorf("%w: contract address not parseable into evm address", ErrInvalidConfig)
	}

	if config.RPC == nil {
		return fmt.Errorf("%w: RPC endpoint required", ErrInvalidConfig)
	}

	if config.ReportOutputPath != nil {
		if _, err := os.Stat(*config.ReportOutputPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: provided report output directory does not exist", ErrInvalidConfig)
		}
	}

	if config.Nodes == nil {
		return fmt.Errorf("%w: number of nodes required", ErrInvalidConfig)
	} else if *config.Nodes <= 0 {
		return fmt.Errorf("%w: must have more than 0 nodes", ErrInvalidConfig)
	}

	if config.Rounds != nil && *config.Rounds < 0 {
		return fmt.Errorf("%w: number of rounds must be 0 or more", ErrInvalidConfig)
	}

	if config.RoundTime != nil && *config.RoundTime < 0 {
		return fmt.Errorf("%w: round time must be 0 or more", ErrInvalidConfig)
	}

	if config.QueryTimeLimit != nil && *config.QueryTimeLimit < 0 {
		return fmt.Errorf("%w: query time limit must be 0 or more", ErrInvalidConfig)
	}

	if config.ObservationTimeLimit != nil && *config.ObservationTimeLimit < 0 {
		return fmt.Errorf("%w: observation time limit must be 0 or more", ErrInvalidConfig)
	}

	if config.ReportTimeLimit != nil && *config.ReportTimeLimit < 0 {
		return fmt.Errorf("%w: report time limit must be 0 or more", ErrInvalidConfig)
	}

	if config.MaxRunTime != nil && *config.MaxRunTime < 0 {
		return fmt.Errorf("%w: max run time limit must be 0 or more", ErrInvalidConfig)
	}

	if config.RoundTime != nil && *config.RoundTime != 0 &&
		((config.QueryTimeLimit != nil && *config.QueryTimeLimit != 0) ||
			(config.ObservationTimeLimit != nil && *config.ObservationTimeLimit != 0) ||
			(config.ReportTimeLimit != nil && *config.ReportTimeLimit != 0)) {
		return fmt.Errorf("%w: round time in conflict with function times (query, observation, report); pick round limits or function limits, not both", ErrInvalidConfig)
	}

	return nil
}

func runFullSimulation(logger *log.Logger, config *SimulatorConfig) error {
	if err := validateSimulatorConfig(config); err != nil {
		return err
	}

	// start the logging
	slogger := NewSimpleLogger(logger.Writer(), Debug)
	defer func() {
		if err := recover(); err != nil {
			slogger.Critical(fmt.Sprint(err), nil)
			debug.PrintStack()
		}
	}()

	w := &logWriter{l: slogger}
	simLogger := log.New(w, "[simulator] ", log.Lshortfile|log.Lmsgprefix)
	lg := log.New(w, "[controller] ", log.Lshortfile|log.Lmsgprefix)

	// create the simulated ocr nodes before creating the controller
	address := common.HexToAddress(*config.ContractAddress)
	receivers := make([]*OCRReceiver, *config.Nodes)
	for i := 0; i < *config.Nodes; i++ {
		receivers[i] = NewOCRReceiver(fmt.Sprintf("node %d", i+1))
	}

	// apply round limits if set by config
	roundTime := time.Duration(0)
	if config.RoundTime != nil {
		roundTime = time.Duration(int64(*config.RoundTime)) * time.Millisecond
	}

	rounds := 0
	if config.Rounds != nil {
		rounds = *config.Rounds
	}

	// create the controller
	controller := NewOCRController(roundTime, rounds, lg, receivers...)

	// apply function time limits if set by config
	if config.QueryTimeLimit != nil {
		controller.QueryTime = time.Duration(int64(*config.QueryTimeLimit)) * time.Millisecond
	}

	if config.ObservationTimeLimit != nil {
		controller.ObservationTime = time.Duration(int64(*config.ObservationTimeLimit)) * time.Millisecond
	}

	if config.ReportTimeLimit != nil {
		controller.ReportTime = time.Duration(int64(*config.ReportTimeLimit)) * time.Millisecond
	}

	// wrap each node in the node simulator
	for i, rec := range receivers {
		l := log.New(w, fmt.Sprintf("[node %d] ", i+1), log.Lshortfile|log.Lmsgprefix)

		// each node has its own rpc connection
		client, err := ethclient.Dial(*config.RPC)
		if err != nil {
			return err
		}

		wrapPluginReceiver(simLogger, controller, rec, makePlugin(address, controller, l, client))
	}

	var ctx context.Context
	var cancel context.CancelFunc

	// apply run time limits to the context
	if config.MaxRunTime != nil && *config.MaxRunTime > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(int64(*config.MaxRunTime))*time.Second)
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

	// write reports on exit if configured to do so
	if config.ReportOutputPath != nil {
		log.Print("writing reports")
		controller.WriteReports(*config.ReportOutputPath)
	}

	return nil
}

func makePlugin(address common.Address, controller *OCRController, logger *log.Logger, client *ethclient.Client) types.ReportingPlugin {
	reg, err := chain.NewEVMRegistryV2_0(address, client)
	if err != nil {
		panic(err)
	}

	config := keepers.ReportingFactoryConfig{
		CacheExpiration:       20 * time.Minute,
		CacheEvictionInterval: 30 * time.Second,
		MaxServiceWorkers:     10 * runtime.GOMAXPROCS(0),
		ServiceQueueLength:    1000,
	}

	factory := keepers.NewReportingPluginFactory(reg, chain.NewEVMReportEncoder(), logger, config)
	plugin, info, err := factory.NewReportingPlugin(types.ReportingPluginConfig{})
	if err != nil {
		panic(err)
	}

	controller.MaxQueryLength = info.Limits.MaxQueryLength
	controller.MaxObservationLength = info.Limits.MaxObservationLength
	controller.MaxReportLength = info.Limits.MaxReportLength

	return plugin
}

func wrapPluginReceiver(logger *log.Logger, controller *OCRController, receiver *OCRReceiver, plugin types.ReportingPlugin) {
	go func(c *OCRController, r *OCRReceiver, p types.ReportingPlugin) {
		for {
			select {
			case call := <-receiver.Init:
				logger.Printf("%s: init call received", receiver.Name)
				q, err := p.Query(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)})
				if err != nil {
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}
				go func() {
					select {
					case c.Queries <- OCRQuery(q):
						logger.Printf("%s: sent query to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.QueryTime > 0 {
							logger.Printf("%s: context ended for query call", receiver.Name)
						}
						return
					}
				}()
			case call := <-receiver.Query:
				logger.Printf("%s: query recieved", receiver.Name)
				o, err := p.Observation(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Query(call.Data))
				if err != nil {
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}
				go func() {
					select {
					case c.Observations <- OCRObservation(o):
						logger.Printf("%s: sent observation to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.ObservationTime > 0 {
							logger.Printf("%s: context ended for observation call", receiver.Name)
						}
						return
					}
				}()
			case call := <-receiver.Observations:
				logger.Printf("%s: observations received", receiver.Name)
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
						logger.Printf("%s: nothing to report; sending empty report", receiver.Name)
						rv = types.Report{}
					}
					select {
					case c.Reports <- OCRReport(rv):
						logger.Printf("%s: sent report to controller", receiver.Name)
						return
					case <-call.Context.Done():
						if controller.ReportTime > 0 {
							logger.Printf("%s: context ended for report call", receiver.Name)
						}
						return
					}
				}()
			case call := <-receiver.Report:
				logger.Printf("%s: report received", receiver.Name)
				b, err := p.ShouldAcceptFinalizedReport(call.Context, types.ReportTimestamp{Round: uint8(call.Round), Epoch: uint32(call.Epoch)}, types.Report(call.Data))
				if err != nil {
					panic(fmt.Sprintf("fatal error in query: %s", err))
				}

				logger.Printf("accept finalized report for round: %d; epoch: %d: %t", call.Round, call.Epoch, b)
			case <-r.Stop:
				return
			}
		}
	}(controller, receiver, plugin)
}
