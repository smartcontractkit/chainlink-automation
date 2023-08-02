package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type MetadataKey string

const (
	BlockHistoryMetadata     MetadataKey = "block history"
	ProposalRecoveryMetadata MetadataKey = "proposed for recovery"
	ProposalSampleMetadata   MetadataKey = "proposed samples"
)

type Metadata struct {
	// dependencies
	blocks *tickers.BlockTicker

	// private values
	mu   sync.RWMutex
	data map[MetadataKey]interface{}

	// internal state
	running atomic.Bool
	stopCh  chan struct{}
}

func NewMetadata(blocks *tickers.BlockTicker) *Metadata {
	return &Metadata{
		blocks: blocks,
		data:   make(map[MetadataKey]interface{}),
		stopCh: make(chan struct{}, 1),
	}
}

func (m *Metadata) Set(key MetadataKey, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
}

func (m *Metadata) Delete(key MetadataKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
}

func (m *Metadata) Get(key MetadataKey) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.data[key]

	return value, ok
}

func (m *Metadata) Start(_ context.Context) error {
	if m.running.Load() {
		return fmt.Errorf("service already running")
	}

	m.running.Store(true)

	for {
		select {
		case h := <-m.blocks.C:
			m.Set(BlockHistoryMetadata, h)
		case <-m.stopCh:
			return nil
		}
	}
}

func (m *Metadata) Close() error {
	if !m.running.Load() {
		return fmt.Errorf("service not running")
	}

	m.stopCh <- struct{}{}
	m.running.Store(false)

	return nil
}
