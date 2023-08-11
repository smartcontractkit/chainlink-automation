package resultstore

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
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
		lggr:          log.New(lggr.Writer(), fmt.Sprintf("[%s | result-store]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
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
		_, ok := s.data[r.WorkID]
		if !ok {
			added++
			s.data[r.WorkID] = result{data: r, addedAt: time.Now()}

			s.lggr.Printf("result added for upkeep id '%s' and trigger '%s'", r.UpkeepID.String(), r.WorkID)
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

		s.lggr.Printf("result removed for key '%s'", id)
	}
}

// View returns a copy of the data in the store.
func (s *resultStore) View() ([]ocr2keepers.CheckResult, error) {
	return s.viewResults(), nil
}

func (s *resultStore) viewResults() []ocr2keepers.CheckResult {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var results []ocr2keepers.CheckResult

	for _, r := range s.data {
		if time.Since(r.addedAt) > storeTTL {
			// expired, we don't want to remove the element here
			// as it requires to acquire a write lock, which slows down the View method
			continue
		}

		results = append(results, r.data)

		s.lggr.Printf("result with upkeep id '%s' and trigger id '%s' viewed", r.data.UpkeepID.String(), r.data.WorkID)
	}
	return results
}

func (s *resultStore) gc() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lggr.Println("garbage collecting result store")

	for k, v := range s.data {
		if time.Since(v.addedAt) > storeTTL {
			delete(s.data, k)
			s.notify(ocr2keepersv3.NotifyOpEvict, v.data)

			s.lggr.Printf("value evicted for upkeep id '%s' and trigger id '%s'", v.data.UpkeepID.String(), v.data.WorkID)
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
