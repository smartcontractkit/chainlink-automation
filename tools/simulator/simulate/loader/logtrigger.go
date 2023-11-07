package loader

import (
	"crypto/sha256"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
)

// LogTriggerLoader ...
type LogTriggerLoader struct {
	// provided dependencies
	progress ProgressTelemetry

	// internal state values
	mu       sync.RWMutex
	triggers map[string][]chain.Log
}

// NewLogTriggerLoader ...
func NewLogTriggerLoader(plan config.SimulationPlan, progress ProgressTelemetry) (*LogTriggerLoader, error) {
	logs, err := chain.GenerateLogTriggers(plan)
	if err != nil {
		return nil, err
	}

	events := make(map[string][]chain.Log)
	for _, logEvt := range logs {
		trigger := logEvt.TriggerAt

		existing, ok := events[trigger.String()]
		if !ok {
			existing = []chain.Log{}
		}

		events[trigger.String()] = append(existing, chain.Log{
			TriggerValue: logEvt.TriggerValue,
		})
	}

	if progress != nil {
		if err := progress.Register(emitLogNamespace, int64(len(logs))); err != nil {
			return nil, err
		}
	}

	return &LogTriggerLoader{
		progress: progress,
		triggers: events,
	}, nil
}

// Load implements the chain.BlockLoaderFunc type and loads log trigger events
// into blocks
func (ltl *LogTriggerLoader) Load(block *chain.Block) {
	ltl.mu.RLock()
	defer ltl.mu.RUnlock()

	if events, ok := ltl.triggers[block.Number.String()]; ok {
		for _, event := range events {
			event.BlockNumber = block.Number
			event.BlockHash = block.Hash
			event.Idx = uint32(len(block.Transactions))
			event.TxHash = sha256.Sum256([]byte(time.Now().Format(time.RFC3339Nano)))

			block.Transactions = append(block.Transactions, event)
		}

		if ltl.progress != nil {
			ltl.progress.Increment(emitLogNamespace, int64(len(events)))
		}
	}
}
