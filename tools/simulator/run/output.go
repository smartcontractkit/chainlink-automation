package run

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/telemetry"
)

type Outputs struct {
	SimulationLog           *log.Logger
	RPCCollector            *telemetry.RPCCollector
	LogCollector            *telemetry.NodeLogCollector
	EventCollector          *telemetry.ContractEventCollector
	simulationLogFileHandle *os.File
}

func (out *Outputs) Close() error {
	var err error

	if out.simulationLogFileHandle != nil {
		err = errors.Join(err, out.simulationLogFileHandle.Close())
	}

	return err
}

func SetupOutput(path string, simulate, verbose bool, plan config.SimulationPlan) (*Outputs, error) {
	if !verbose {
		logger := log.New(io.Discard, "", 0)

		return &Outputs{
			SimulationLog:  logger,
			RPCCollector:   telemetry.NewNodeRPCCollector("", false),
			LogCollector:   telemetry.NewNodeLogCollector("", false),
			EventCollector: telemetry.NewContractEventCollector(logger),
		}, nil
	}

	var (
		logger *log.Logger
		lggF   *os.File
		err    error
	)

	// always setup the output directory
	// the simulation will write to this directory and charts will read
	err = os.MkdirAll(path, 0750)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}

	// only when running a simulation is the simulation log needed
	// if a simulation has already been run, don't write out the current
	// simulation plan
	if simulate {
		logger, lggF, err = openSimulationLog(path)
		if err != nil {
			return nil, err
		}

		if err := saveSimulationPlanToOutput(path, plan); err != nil {
			return nil, err
		}
	}

	return &Outputs{
		SimulationLog:           logger,
		RPCCollector:            telemetry.NewNodeRPCCollector(path, true),
		LogCollector:            telemetry.NewNodeLogCollector(path, true),
		EventCollector:          telemetry.NewContractEventCollector(logger),
		simulationLogFileHandle: lggF,
	}, nil
}

func saveSimulationPlanToOutput(path string, plan config.SimulationPlan) error {
	filename := fmt.Sprintf("%s/simulation_plan.json", path)
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	f, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return fmt.Errorf("failed to open simulation plan file (%s): %v", filename, err)
	}

	defer f.Close()

	b, err := plan.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode simulation_plan: %w", err)
	}

	l, err := f.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write encoded simulation plan to file (%s): %w", filename, err)
	}

	if l != len(b) {
		return fmt.Errorf("failed to write encoded simulation plan to file (%s): not all bytes written", filename)
	}

	return nil
}

func openSimulationLog(path string) (*log.Logger, *os.File, error) {
	filename := fmt.Sprintf("%s/simulation.log", path)
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	f, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file (%s): %v", filename, err)
	}

	return log.New(f, "", log.LstdFlags), f, nil
}
