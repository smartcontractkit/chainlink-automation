package proposalqueue

import (
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

var (
	DefaultExpiration = 24 * time.Hour
)

type proposalQueueRecord struct {
	proposal ocr2keepers.CoordinatedProposal
	// visited is true if the record was already dequeued
	removed bool
	// createdAt is the first time the proposal was seen by the queue
	createdAt time.Time
}

func (r proposalQueueRecord) expired(now time.Time, expr time.Duration) bool {
	return now.Sub(r.createdAt) > expr
}

type proposalQueue struct {
	lock    sync.RWMutex
	records map[string]proposalQueueRecord

	typeGetter ocr2keepers.UpkeepTypeGetter
}

var _ ocr2keepers.ProposalQueue = &proposalQueue{}

func New(typeGetter ocr2keepers.UpkeepTypeGetter) *proposalQueue {
	return &proposalQueue{
		records:    map[string]proposalQueueRecord{},
		typeGetter: typeGetter,
	}
}

func (pq *proposalQueue) Enqueue(newProposals ...ocr2keepers.CoordinatedProposal) error {
	pq.lock.Lock()
	defer pq.lock.Unlock()

	for _, p := range newProposals {
		if _, ok := pq.records[p.WorkID]; ok {
			continue
		}
		pq.records[p.WorkID] = proposalQueueRecord{
			proposal:  p,
			createdAt: time.Now(),
		}
	}

	return nil
}

func (pq *proposalQueue) Dequeue(t ocr2keepers.UpkeepType, n int) ([]ocr2keepers.CoordinatedProposal, error) {
	pq.lock.Lock()
	defer pq.lock.Unlock()

	var proposals []ocr2keepers.CoordinatedProposal
	for _, record := range pq.records {
		if record.expired(time.Now(), DefaultExpiration) {
			delete(pq.records, record.proposal.WorkID)
			continue
		}
		if record.removed {
			continue
		}
		if pq.typeGetter(record.proposal.UpkeepID) == t {
			proposals = append(proposals, record.proposal)
		}
	}
	if len(proposals) < n {
		n = len(proposals)
	}
	// limit the number of proposals returned
	proposals = proposals[:n]
	// mark results as removed
	for _, p := range proposals {
		proposal := pq.records[p.WorkID]
		proposal.removed = true
		pq.records[p.WorkID] = proposal
	}

	return proposals, nil
}

func (pq *proposalQueue) Size() int {
	pq.lock.RLock()
	defer pq.lock.RUnlock()

	now := time.Now()
	size := 0

	for _, record := range pq.records {
		if record.removed || record.expired(now, DefaultExpiration) {
			continue
		}
		size++
	}

	return size
}
