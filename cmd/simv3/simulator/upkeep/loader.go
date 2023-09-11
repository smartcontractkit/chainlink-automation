package upkeep

import (
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
)

// UpkeepConfigLoader provides upkeep configurations to a block broadcaster. Use
// this loader to introduce upkeeps or change upkeep configs at specific block
// numbers.
type UpkeepConfigLoader struct {
	mu           sync.RWMutex
	conditionals []chain.SimulatedUpkeep
	events       map[string][]interface{}
}

// NewUpkeepConfigLoader ...
func NewUpkeepConfigLoader(rb config.RunBook) (*UpkeepConfigLoader, error) {
	// generate conditionals
	conditionals, err := GenerateConditionals(rb)
	if err != nil {
		return nil, err
	}

	logTriggered, err := GenerateLogTriggeredUpkeeps(rb)
	if err != nil {
		return nil, err
	}

	allUpkeeps := append(conditionals, logTriggered...)

	// TODO: create more event types (create, cancel, pause, etc)
	// the only currently supported type is create and will create on the genesis
	// block
	events := make(map[string][]interface{})
	for _, upkeep := range allUpkeeps {
		evts, ok := events[rb.BlockCadence.Genesis.String()]
		if !ok {
			evts = []interface{}{}
		}

		events[rb.BlockCadence.Genesis.String()] = append(evts, chain.UpkeepCreatedTransaction{
			Upkeep: upkeep,
		})
	}

	return &UpkeepConfigLoader{
		conditionals: conditionals,
		events:       events,
	}, nil
}

// Load implements the chain.BlockLoaderFunc type and loads configured upkeep
// events into blocks.
func (ucl *UpkeepConfigLoader) Load(block *chain.Block) {
	ucl.mu.RLock()
	defer ucl.mu.RUnlock()

	if events, ok := ucl.events[block.Number.String()]; ok {
		block.Transactions = append(block.Transactions, events...)
	}
}

// LogTriggerLoader ...
type LogTriggerLoader struct {
	mu       sync.RWMutex
	triggers map[string][]interface{}
}

// NewLogTriggerLoader ...
func NewLogTriggerLoader(rb config.RunBook) (*LogTriggerLoader, error) {
	logs, err := GenerateLogTriggers(rb)
	if err != nil {
		return nil, err
	}

	events := make(map[string][]interface{})
	for _, logEvt := range logs {
		for _, trigger := range logEvt.TriggerAt {
			existing, ok := events[trigger.String()]
			if !ok {
				existing = []interface{}{}
			}

			events[trigger.String()] = append(existing, chain.Log{
				TriggerValue: logEvt.TriggerValue,
			})
		}
	}

	return &LogTriggerLoader{
		triggers: events,
	}, nil
}

// Load implements the chain.BlockLoaderFunc type and loads log trigger events
// into blocks
func (ltl *LogTriggerLoader) Load(block *chain.Block) {
	ltl.mu.RLock()
	defer ltl.mu.RUnlock()

	if events, ok := ltl.triggers[block.Number.String()]; ok {
		block.Transactions = append(block.Transactions, events...)
	}
}
