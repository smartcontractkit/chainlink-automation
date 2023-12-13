package telemetry

import (
	"fmt"
	"io"
	"log"
	"sync"
)

type ContractEventCollector struct {
	baseCollector

	// dependencies
	logger *log.Logger

	// internal state properties
	mu    sync.RWMutex
	nodes map[string]*WrappedContractCollector
}

func NewContractEventCollector(logger *log.Logger) *ContractEventCollector {
	return &ContractEventCollector{
		baseCollector: baseCollector{
			t:        NodeLogType,
			io:       []io.WriteCloser{},
			ioLookup: make(map[string]int),
		},
		logger: log.New(logger.Writer(), "[contract-event-collector]", log.Ldate|log.Ltime|log.Lshortfile),
		nodes:  make(map[string]*WrappedContractCollector),
	}
}

func (c *ContractEventCollector) ContractEventCollectorNode(node string) *WrappedContractCollector {
	if wc, ok := c.nodes[node]; ok {
		return wc
	}

	panic("node not available")
}

func (c *ContractEventCollector) Data() (map[string]int, map[string][]string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// keyChecks is per id per block
	allKeyChecks := make(map[string]int)

	// idLookup is id checked in blocks
	allKeyIDLookup := make(map[string][]string)

	for _, node := range c.nodes {
		for key, value := range node.keyChecks {
			v, ok := allKeyChecks[key]
			if !ok {
				allKeyChecks[key] = value
			} else {
				allKeyChecks[key] += v
			}
		}

		for key, lookup := range node.keyIDLookup {
			v, ok := allKeyIDLookup[key]
			if !ok {
				allKeyIDLookup[key] = lookup
			} else {
				for _, ls := range lookup {
					found := false
					for _, ex := range v {
						if ls == ex {
							found = true
							break
						}
					}

					if !found {
						v = append(v, ls)
					}
				}
				allKeyIDLookup[key] = v
			}
		}
	}

	return allKeyChecks, allKeyIDLookup
}

func (c *ContractEventCollector) AddNode(node string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	wc := &WrappedContractCollector{
		name:        node,
		logger:      c.logger,
		keyChecks:   make(map[string]int),
		keyIDLookup: make(map[string][]string),
	}

	c.nodes[node] = wc

	return nil
}

type WrappedContractCollector struct {
	// provided dependencies
	name   string
	logger *log.Logger

	// internal state properties
	mu          sync.Mutex
	keyChecks   map[string]int
	keyIDLookup map[string][]string
}

func (wc *WrappedContractCollector) CheckID(upkeepID string, number uint64, _ [32]byte) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if _, ok := wc.keyChecks[upkeepID]; !ok {
		wc.keyChecks[upkeepID] = 0
	}

	wc.keyChecks[upkeepID]++

	strBlock := fmt.Sprintf("%d", number)

	if blocks, ok := wc.keyIDLookup[upkeepID]; ok {
		var found bool

		for _, block := range blocks {
			if block == strBlock {
				found = true
			}
		}

		if !found {
			wc.keyIDLookup[upkeepID] = append(blocks, strBlock)
		}

		return
	}

	wc.keyIDLookup[upkeepID] = []string{strBlock}
}
