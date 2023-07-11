package run

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/telemetry"
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

func SetupOutput(path string, simulate bool, runbook config.RunBook) (*Outputs, error) {
	var (
		lgg  *log.Logger
		lggF *os.File
		err  error
	)

	// always setup the output directory
	// the simulation will write to this directory and charts will read
	err = os.MkdirAll(path, 0750)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, err
	}

	// only when running a simulation is the simulation log needed
	// if a simulation has already been run, don't write out the current runbook
	if simulate {
		lgg, lggF, err = openSimulationLog(path)
		if err != nil {
			return nil, err
		}

		if err := saveRunbookToOutput(path, runbook); err != nil {
			return nil, err
		}
	}

	return &Outputs{
		SimulationLog:           lgg,
		RPCCollector:            telemetry.NewNodeRPCCollector(path),
		LogCollector:            telemetry.NewNodeLogCollector(path),
		EventCollector:          telemetry.NewContractEventCollector(path, nil),
		simulationLogFileHandle: lggF,
	}, nil
}

func saveRunbookToOutput(path string, rb config.RunBook) error {
	filename := fmt.Sprintf("%s/runbook.json", path)
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	f, err := os.OpenFile(filename, flags, 0666)
	if err != nil {
		return fmt.Errorf("failed to open runbook file (%s): %v", filename, err)
	}

	defer f.Close()

	b, err := rb.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode runbook: %w", err)
	}

	l, err := f.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write encoded runbook to file (%s): %w", filename, err)
	}

	if l != len(b) {
		return fmt.Errorf("failed to write encoded runbook to file (%s): not all bytes written", filename)
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
