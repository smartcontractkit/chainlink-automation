package stores

import (
	"context"
	"fmt"
	"sort"
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

var (
	timeFn = time.Now
)

type expiringRecord struct {
	createdAt time.Time
	proposal  types.CoordinatedProposal
}

func (r expiringRecord) expired(expr time.Duration) bool {
	return time.Since(r.createdAt) > expr
}

type metadataStore struct {
	blocks               *tickers.BlockTicker
	blockHistory         types.BlockHistory
	blockHistoryMutex    sync.RWMutex
	conditionalProposals orderedMap
	conditionalMutex     sync.RWMutex
	logRecoveryProposals orderedMap
	logRecoveryMutex     sync.RWMutex
	running              atomic.Bool
	stopCh               chan struct{}

	typeGetter types.UpkeepTypeGetter
}

func NewMetadataStore(blocks *tickers.BlockTicker, typeGetter types.UpkeepTypeGetter) *metadataStore {
	return &metadataStore{
		blocks:               blocks,
		blockHistory:         types.BlockHistory{},
		conditionalProposals: newOrderedMap(),
		logRecoveryProposals: newOrderedMap(),
		stopCh:               make(chan struct{}, 1),
		typeGetter:           typeGetter,
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

func (m *metadataStore) AddProposals(proposals ...types.CoordinatedProposal) {
	for _, proposal := range proposals {
		switch m.typeGetter(proposal.UpkeepID) {
		case types.LogTrigger:
			m.AddLogRecoveryProposal(proposal)
		case types.ConditionTrigger:
			m.AddConditionalProposal(proposal)
		}
	}
}

func (m *metadataStore) ViewProposals(utype types.UpkeepType) []types.CoordinatedProposal {
	switch utype {
	case types.LogTrigger:
		return m.ViewLogRecoveryProposal()
	case types.ConditionTrigger:
		return m.ViewConditionalProposal()
	default:
		return nil
	}
}

func (m *metadataStore) RemoveProposals(proposals ...types.CoordinatedProposal) {
	for _, proposal := range proposals {
		switch m.typeGetter(proposal.UpkeepID) {
		case types.LogTrigger:
			m.RemoveLogRecoveryProposal(proposal)
		case types.ConditionTrigger:
			m.RemoveConditionalProposal(proposal)
		}
	}
}

func (m *metadataStore) AddLogRecoveryProposal(proposals ...types.CoordinatedProposal) {
	m.logRecoveryMutex.Lock()
	defer m.logRecoveryMutex.Unlock()

	for _, proposal := range proposals {
		m.logRecoveryProposals.Add(proposal.WorkID, expiringRecord{
			createdAt: timeFn(),
			proposal:  proposal,
		})
	}
}

func (m *metadataStore) ViewLogRecoveryProposal() []types.CoordinatedProposal {
	m.logRecoveryMutex.RLock()
	defer m.logRecoveryMutex.RUnlock()

	res := make([]types.CoordinatedProposal, 0)

	for _, key := range m.logRecoveryProposals.Keys() {
		record := m.logRecoveryProposals.Get(key)
		if record.expired(logRecoveryExpiry) {
			m.logRecoveryProposals.Delete(key)
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
		m.logRecoveryProposals.Delete(proposal.WorkID)
	}
}

func (m *metadataStore) AddConditionalProposal(proposals ...types.CoordinatedProposal) {
	m.conditionalMutex.Lock()
	defer m.conditionalMutex.Unlock()

	for _, proposal := range proposals {
		m.conditionalProposals.Add(proposal.WorkID, expiringRecord{
			createdAt: timeFn(),
			proposal:  proposal,
		})
	}
}

func (m *metadataStore) ViewConditionalProposal() []types.CoordinatedProposal {
	m.conditionalMutex.RLock()
	defer m.conditionalMutex.RUnlock()

	res := make([]types.CoordinatedProposal, 0)

	for _, key := range m.conditionalProposals.Keys() {
		record := m.conditionalProposals.Get(key)
		if record.expired(conditionalExpiry) {
			m.conditionalProposals.Delete(key)
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
		m.conditionalProposals.Delete(proposal.WorkID)
	}
}

func (m *metadataStore) Start(ctx context.Context) error {
	if m.running.Load() {
		return fmt.Errorf("service already running")
	}

	m.running.Store(true)

	for {
		select {
		case h := <-m.blocks.C:
			m.SetBlockHistory(h)
		case <-ctx.Done():
			return m.Close()
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

func newOrderedMap() orderedMap {
	return orderedMap{
		keys:   []string{},
		values: map[string]expiringRecord{},
	}
}

type orderedMap struct {
	keys   []string
	values map[string]expiringRecord
}

func (m *orderedMap) Add(key string, value expiringRecord) {
	if _, ok := m.values[key]; ok {
		m.values[key] = value
	} else {
		m.keys = append(m.keys, key)
		m.values[key] = value
	}
}

func (m *orderedMap) Get(key string) expiringRecord {
	return m.values[key]
}

func (m *orderedMap) Keys() []string {
	sort.Strings(m.keys)
	return m.keys
}

func (m *orderedMap) Delete(key string) {
	delete(m.values, key)
	for i, v := range m.keys {
		if v == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			break
		}
	}
}
