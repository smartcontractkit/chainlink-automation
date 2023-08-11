package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	logRecoveryExpiry = 24 * time.Hour
	conditionalExpiry = 24 * time.Hour
)

//go:generate mockery --name MetadataStore --structname MockMetadataStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/store" --case underscore --filename metadatastore.generated.go
type MetadataStore interface {
	SetBlockHistory(types.BlockHistory)
	GetBlockHistory() types.BlockHistory

	AddLogRecoveryProposal(...types.CoordinatedProposal)
	ViewLogRecoveryProposal() []types.CoordinatedProposal
	RemoveLogRecoveryProposal(...types.CoordinatedProposal)

	AddConditionalProposal(...types.CoordinatedProposal)
	ViewConditionalProposal() []types.CoordinatedProposal
	RemoveConditionalProposal(...types.CoordinatedProposal)

	Start(context.Context) error
	Close() error
}

type expiringRecord struct {
	createdAt time.Time
	proposal  types.CoordinatedProposal
}

func (r expiringRecord) expired(expr time.Duration) bool {
	return time.Now().Sub(r.createdAt) > expr
}

type metadataStore struct {
	blocks               *tickers.BlockTicker
	blockHistory         types.BlockHistory
	blockHistoryMutex    sync.RWMutex
	conditionalProposals map[string]expiringRecord
	conditionalMutex     sync.RWMutex
	logRecoveryProposals map[string]expiringRecord
	logRecoveryMutex     sync.RWMutex
	running              atomic.Bool
	stopCh               chan struct{}
}

func NewMetadataStore(blocks *tickers.BlockTicker) *metadataStore {
	return &metadataStore{
		blocks:               blocks,
		blockHistory:         types.BlockHistory{},
		conditionalProposals: map[string]expiringRecord{},
		logRecoveryProposals: map[string]expiringRecord{},
		stopCh:               make(chan struct{}, 1),
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

func (m *metadataStore) AddLogRecoveryProposal(proposals ...types.CoordinatedProposal) {
	m.logRecoveryMutex.Lock()
	defer m.logRecoveryMutex.Unlock()

	for _, proposal := range proposals {
		m.logRecoveryProposals[proposal.WorkID] = expiringRecord{
			createdAt: time.Now(),
			proposal:  proposal,
		}
	}
}

func (m *metadataStore) ViewLogRecoveryProposal() []types.CoordinatedProposal {
	m.logRecoveryMutex.RLock()
	defer m.logRecoveryMutex.RUnlock()

	res := make([]types.CoordinatedProposal, 0)

	for key, record := range m.logRecoveryProposals {
		if record.expired(logRecoveryExpiry) {
			delete(m.logRecoveryProposals, key)
		} else {
			res = append(res, record.proposal)
		}
	}

	return res

}

func (m *metadataStore) RemoveLogRecoveryProposal(proposals ...types.CoordinatedProposal) {
	m.logRecoveryMutex.Lock()
	defer m.logRecoveryMutex.Unlock()

	for _, proposal := range proposals {
		delete(m.logRecoveryProposals, proposal.WorkID)
	}
}

func (m *metadataStore) AddConditionalProposal(proposals ...types.CoordinatedProposal) {
	m.conditionalMutex.Lock()
	defer m.conditionalMutex.Unlock()

	for _, proposal := range proposals {
		m.conditionalProposals[proposal.WorkID] = expiringRecord{
			createdAt: time.Now(),
			proposal:  proposal,
		}
	}
}

func (m *metadataStore) ViewConditionalProposal() []types.CoordinatedProposal {
	m.conditionalMutex.RLock()
	defer m.conditionalMutex.RUnlock()

	res := make([]types.CoordinatedProposal, 0)

	for key, record := range m.conditionalProposals {
		if record.expired(conditionalExpiry) {
			delete(m.conditionalProposals, key)
		} else {
			res = append(res, record.proposal)
		}
	}

	return res

}

func (m *metadataStore) RemoveConditionalProposal(proposals ...types.CoordinatedProposal) {
	m.conditionalMutex.Lock()
	defer m.conditionalMutex.Unlock()

	for _, proposal := range proposals {
		delete(m.conditionalProposals, proposal.WorkID)
	}
}

func (m *metadataStore) Start(_ context.Context) error {
	if m.running.Load() {
		return fmt.Errorf("service already running")
	}

	m.running.Store(true)

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

	m.stopCh <- struct{}{}
	m.running.Store(false)

	return nil
}
