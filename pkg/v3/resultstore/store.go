package resultstore

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

// TODO: make these configurable?
var (
	notifyQBufferSize = 128
	storeTTL          = time.Minute
	gcInterval        = time.Minute * 5
)

// result is an internal representation of a check result, with added time for TTL.
type result struct {
	data    ocr2keepers.CheckResult
	addedAt time.Time
}

// resultStore implements ResultStore.
type resultStore struct {
	lggr *log.Logger

	close chan bool

	data map[string]result
	lock sync.RWMutex

	notifications chan ocr2keepersv3.Notification
}

func New(lggr *log.Logger) *resultStore {
	return &resultStore{
		lggr:          lggr,
		close:         make(chan bool, 1),
		data:          make(map[string]result),
		lock:          sync.RWMutex{},
		notifications: make(chan ocr2keepersv3.Notification, notifyQBufferSize),
	}
}

// Start starts the store, it spins up a goroutine that runs the garbage collector every gcInterval.
func (s *resultStore) Start(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	s.lggr.Println("starting result store")

	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.gc()
		case <-ctx.Done():
			s.lggr.Println("result store context done, stopping gc")
			return nil
		case <-s.close:
			s.lggr.Println("result store close signal received, stopping gc")
			return nil
		}
	}
}

func (s *resultStore) Close() error {
	s.close <- true
	return nil
}

// Notifications returns a channel that can be used to receive notifications about evicted/removed items in the store.
func (s *resultStore) Notifications() <-chan ocr2keepersv3.Notification {
	return s.notifications
}

// Add adds element/s to the store.
func (s *resultStore) Add(results ...ocr2keepers.CheckResult) {
	s.lock.Lock()
	defer s.lock.Unlock()

	added := 0
	for _, r := range results {
		id := r.Payload.ID
		_, ok := s.data[id]
		if !ok {
			added++
			s.data[id] = result{data: r, addedAt: time.Now()}
		}
		// if the element is already exists, we do noting
	}
}

// Remove removes element/s from the store.
func (s *resultStore) Remove(ids ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, id := range ids {
		s.remove(id)
	}
}

// View returns a copy of the data in the store.
// It accepts filters that can be used to prepare the results view.
// NOTE: we apply filters while holding the read lock, these functions must not block.
func (s *resultStore) View(opts ...ocr2keepersv3.ViewOpt) ([]ocr2keepers.CheckResult, error) {
	filters, comparators, limit := ocr2keepersv3.ViewOpts(opts).Apply()

	results, limit := s.viewResults(limit, filters, comparators)
	s.orderResults(results, comparators)

	if limit > len(results) {
		limit = len(results)
	}

	return results[:limit], nil
}

func (s *resultStore) orderResults(results []ocr2keepers.CheckResult, comparators []ocr2keepersv3.ResultComparator) {
	if len(comparators) > 0 {
		sort.SliceStable(results, func(i, j int) bool {
			for _, comparator := range comparators {
				if !comparator(results[i], results[j]) {
					return false
				}
			}
			return true
		})
	}
}

func (s *resultStore) viewResults(
	limit int,
	filters []ocr2keepersv3.ResultFilter,
	comparators []ocr2keepersv3.ResultComparator,
) ([]ocr2keepers.CheckResult, int) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var results []ocr2keepers.CheckResult
	if limit == 0 {
		limit = len(s.data)
	}

resultLoop:
	for _, r := range s.data {
		if time.Since(r.addedAt) > storeTTL {
			// expired, we don't want to remove the element here
			// as it requires to acquire a write lock, which slows down the View method
			continue
		}
		// apply filters
		for _, filter := range filters {
			if !filter(r.data) {
				continue resultLoop
			}
		}
		results = append(results, r.data)
		// if we reached the limit and there are no comparators, we can stop here
		if len(results) == limit && len(comparators) == 0 {
			break
		}
	}
	return results, limit
}

func (s *resultStore) gc() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lggr.Println("garbage collecting result store")

	for k, v := range s.data {
		if time.Since(v.addedAt) > storeTTL {
			delete(s.data, k)
			s.notify(ocr2keepersv3.NotifyOpEvict, v.data)
		}
	}
}

// notify writes to the notifications channel.
// NOTE: we drop notifications in case the channel is full
func (s *resultStore) notify(op ocr2keepersv3.NotifyOp, data ocr2keepers.CheckResult) {
	select {
	case s.notifications <- ocr2keepersv3.Notification{
		Op:   op,
		Data: data,
	}:
	default:
		s.lggr.Println("notifications queue is full, dropping result")
	}
}

// remove removes an element from the store.
// NOTE: not thread safe, must be called with lock held
func (s *resultStore) remove(id string) {
	v, ok := s.data[id]
	if !ok {
		return
	}
	delete(s.data, id)
	s.notify(ocr2keepersv3.NotifyOpRemove, v.data)
}