package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	proposalLogExpiry          = 24 * time.Hour
	proposalLogCleanupInterval = time.Hour
)

type metadataStore struct {
	blocks                     *tickers.BlockTicker
	blockHistory               types.BlockHistory
	blockHistoryMutex          sync.RWMutex
	proposalLogRecovery        *util.Cache[types.CoordinatedProposal]
	proposalLogRecoveryCleaner *util.IntervalCacheCleaner[types.CoordinatedProposal]
	proposalConditional        []types.CoordinatedProposal
	conditionalMutex           sync.RWMutex
	running                    atomic.Bool
	stopCh                     chan struct{}
}

func NewMetadataStore(blocks *tickers.BlockTicker) *metadataStore {
	return &metadataStore{
		blocks:                     blocks,
		blockHistory:               types.BlockHistory{},
		proposalLogRecovery:        util.NewCache[types.CoordinatedProposal](proposalLogExpiry),
		proposalLogRecoveryCleaner: util.NewIntervalCacheCleaner[types.CoordinatedProposal](proposalLogCleanupInterval),
		proposalConditional:        []types.CoordinatedProposal{},
		stopCh:                     make(chan struct{}, 1),
	}
}

func (m *metadataStore) SetBlockHistory(blockHistory types.BlockHistory) {
	m.blockHistoryMutex.Lock()
	defer m.blockHistoryMutex.Unlock()

	m.blockHistory = blockHistory
}

func (m *metadataStore) GetBlockHistory() types.BlockHistory {
	m.blockHistoryMutex.RLock()
	defer m.blockHistoryMutex.RUnlock()

	return m.blockHistory
}

func (m *metadataStore) GetProposalLogRecovery(key string) (types.CoordinatedProposal, bool) {
	return m.proposalLogRecovery.Get(key)
}

func (m *metadataStore) GetProposalLogRecoveryKeys() []string {
	return m.proposalLogRecovery.Keys()
}

func (m *metadataStore) RemoveProposalLogRecovery(key string) {
	m.proposalLogRecovery.Delete(key)
}

func (m *metadataStore) ClearAllProposalLogRecovery() {
	m.proposalLogRecovery.ClearAll()
}

func (m *metadataStore) ClearExpiredProposalLogRecovery() {
	m.proposalLogRecovery.ClearExpired()
}

func (m *metadataStore) AppendProposalConditional(proposals ...types.CoordinatedProposal) {
	m.conditionalMutex.Lock()
	defer m.conditionalMutex.Unlock()

	m.proposalConditional = append(m.proposalConditional, proposals...)
}

func (m *metadataStore) GetProposalConditional() []types.CoordinatedProposal {
	m.conditionalMutex.RLock()
	defer m.conditionalMutex.RUnlock()

	return m.proposalConditional
}

func (m *metadataStore) RemoveProposalConditional(proposals ...types.CoordinatedProposal) []types.CoordinatedProposal {
	m.conditionalMutex.Lock()
	defer m.conditionalMutex.Unlock()

	proposalsToRemove := make(map[types.CoordinatedProposal]bool)

	for _, proposal := range proposals {
		proposalsToRemove[proposal] = true
	}

	var updatedProposals []types.CoordinatedProposal
	for _, proposal := range m.proposalConditional {
		if !proposalsToRemove[proposal] {
			updatedProposals = append(updatedProposals, proposal)
		}
	}

	m.proposalConditional = updatedProposals
	return m.proposalConditional
}

func (m *metadataStore) Start(_ context.Context) error {
	if m.running.Load() {
		return fmt.Errorf("service already running")
	}

	m.running.Store(true)

	go m.proposalLogRecoveryCleaner.Run(m.proposalLogRecovery)

	for {
		select {
		case h := <-m.blocks.C:
			m.SetBlockHistory(h)
		case <-m.stopCh:
			return nil
		}
	}
}

func (m *metadataStore) Close() error {
	if !m.running.Load() {
		return fmt.Errorf("service not running")
	}

	m.proposalLogRecoveryCleaner.Stop()
	m.stopCh <- struct{}{}
	m.running.Store(false)

	return nil
}
