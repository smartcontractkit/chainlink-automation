package simulators

import (
	"context"
	"math/big"
	"sort"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (ct *SimulatedContract) PerformLogs(ctx context.Context) ([]types.PerformLog, error) {
	logs := []types.PerformLog{}
	if ct.lastBlock == nil {
		return logs, nil
	}

	keys := ct.perLogs.Keys(100)
	for _, key := range keys {
		lgs, ok := ct.perLogs.Get(key)
		if ok {
			for _, log := range lgs {
				trBlock, trOk := new(big.Int).SetString(log.TransmitBlock.String(), 10)
				if trOk {
					log.Confirmations = new(big.Int).Sub(ct.lastBlock, trBlock).Int64()
					logs = append(logs, log)
				}
			}
		}
	}

	return logs, nil
}

func (ct *SimulatedContract) StaleReportLogs(ctx context.Context) ([]types.StaleReportLog, error) {
	// Not implemented in simulated contract yet
	return nil, nil
}

type sortedKeyMap[T any] struct {
	mu     sync.RWMutex
	values map[string]T
	keys   []string
}

func newSortedKeyMap[T any]() *sortedKeyMap[T] {
	return &sortedKeyMap[T]{
		values: make(map[string]T),
		keys:   make([]string, 0),
	}
}

func (m *sortedKeyMap[T]) Set(key string, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.values[key]
	if !ok {
		m.keys = append(m.keys, key)
		sort.Strings(m.keys)
	}

	m.values[key] = value
}

func (m *sortedKeyMap[T]) Get(key string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.values[key]
	if ok {
		return v, ok
	}

	return getZero[T](), false
}

func (m *sortedKeyMap[T]) Keys(l int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if l > len(m.keys) {
		l = len(m.keys)
	}

	// keys are sorted ascending by block number
	// only return the last 'l' keys
	keys := make([]string, l)
	// loop starting at 1 so the first insert can be l-1, or the last item
	for i := 1; i <= l; i++ {
		keys[i-1] = m.keys[l-i]
	}

	return keys
}

func getZero[T any]() T {
	var result T
	return result
}
