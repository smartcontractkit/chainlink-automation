package loader

import (
	"sync"

	"github.com/smartcontractkit/ocr2keepers/tools/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/tools/simulator/simulate/chain"
)

const (
	CreateUpkeepNamespace = "Emitting create upkeep transactions"
	emitLogNamespace      = "Emitting log events"
)

// UpkeepConfigLoader provides upkeep configurations to a block broadcaster. Use
// this loader to introduce upkeeps or change upkeep configs at specific block
// numbers.
type UpkeepConfigLoader struct {
	// provided dependencies
	progress ProgressTelemetry

	// internal state values
	mu     sync.RWMutex
	create map[string][]chain.UpkeepCreatedTransaction
}

// NewUpkeepConfigLoader ...
func NewUpkeepConfigLoader(plan config.SimulationPlan, progress ProgressTelemetry) (*UpkeepConfigLoader, error) {
	// combine all upkeeps together for transmit
	allUpkeeps, err := chain.GenerateAllUpkeeps(plan)
	if err != nil {
		return nil, err
	}

	create := make(map[string][]chain.UpkeepCreatedTransaction)
	for _, upkeep := range allUpkeeps {
		evts, ok := create[upkeep.CreateInBlock.String()]
		if !ok {
			evts = []chain.UpkeepCreatedTransaction{}
		}

		create[upkeep.CreateInBlock.String()] = append(evts, chain.UpkeepCreatedTransaction{
			Upkeep: upkeep,
		})
	}

	if progress != nil {
		if err := progress.Register(CreateUpkeepNamespace, int64(len(allUpkeeps))); err != nil {
			return nil, err
		}
	}

	return &UpkeepConfigLoader{
		create:   create,
		progress: progress,
	}, nil
}

// Load implements the chain.BlockLoaderFunc type and loads configured upkeep
// events into blocks.
func (ucl *UpkeepConfigLoader) Load(block *chain.Block) {
	ucl.mu.RLock()
	defer ucl.mu.RUnlock()

	if events, ok := ucl.create[block.Number.String()]; ok {
		for _, event := range events {
			block.Transactions = append(block.Transactions, event)
		}

		if ucl.progress != nil {
			ucl.progress.Increment(CreateUpkeepNamespace, int64(len(events)))
		}
	}
}
