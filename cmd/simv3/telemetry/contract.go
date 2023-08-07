package telemetry

import (
	"io"
	"sync"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type ContractEventCollector struct {
	Splitter KeySplitter
	baseCollector
	filePath string
	nodes    map[string]*WrappedContractCollector
}

func NewContractEventCollector(path string, s KeySplitter) *ContractEventCollector {
	return &ContractEventCollector{
		baseCollector: baseCollector{
			t:        NodeLogType,
			io:       []io.WriteCloser{},
			ioLookup: make(map[string]int),
		},
		Splitter: s,
		filePath: path,
		nodes:    make(map[string]*WrappedContractCollector),
	}
}

func (c *ContractEventCollector) ContractEventCollectorNode(node string) *WrappedContractCollector {
	wc, ok := c.nodes[node]
	if !ok {
		panic("node not available")
	}

	return wc
}

func (c *ContractEventCollector) Data() (map[string]int, map[string][]string) {
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
	wc := &WrappedContractCollector{
		Splitter:    c.Splitter,
		keyChecks:   make(map[string]int),
		keyIDLookup: make(map[string][]string),
	}

	c.nodes[node] = wc

	return nil
}

type KeySplitter interface {
	// SplitUpkeepKey ...
	SplitUpkeepKey(ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error)
}

type WrappedContractCollector struct {
	Splitter    KeySplitter
	mu          sync.Mutex
	keyChecks   map[string]int
	keyIDLookup map[string][]string
}

func (wc *WrappedContractCollector) CheckID(id string, block ocr2keepers.BlockKey) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	_, ok := wc.keyChecks[id]
	if !ok {
		wc.keyChecks[id] = 0
	}
	wc.keyChecks[id]++

	val, ok := wc.keyIDLookup[id]
	if !ok {
		wc.keyIDLookup[id] = []string{string(block)}
	} else {
		var found bool
		for _, v := range val {
			if v == string(block) {
				found = true
			}
		}

		if !found {
			wc.keyIDLookup[id] = append(val, string(block))
		}
	}
}
