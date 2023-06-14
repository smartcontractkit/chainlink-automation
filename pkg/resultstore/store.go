package resultstore

import (
	"log"
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type Filter func(res ocr2keepers.CheckResult) bool

type NotifyOp uint8

const (
	NotifyOpNil NotifyOp = iota
	NotifyOpEvict
	NotifyOpRemove
)

type Notification struct {
	Op   NotifyOp
	Data ocr2keepers.CheckResult
}

type ResultStore interface {
	Add(...ocr2keepers.CheckResult)
	Remove(...string)
	View(...Filter) ([]ocr2keepers.CheckResult, error)
}

var (
	notifyQBufferSize = 128
	storeTTL          = time.Minute
)

type element struct {
	data    ocr2keepers.CheckResult
	addedAt time.Time
}

type resultStore struct {
	lggr *log.Logger

	data map[string]element
	lock sync.RWMutex

	notifications chan Notification
}

func NewResultStore(lggr *log.Logger) *resultStore {
	return &resultStore{
		lggr:          lggr,
		data:          make(map[string]element),
		lock:          sync.RWMutex{},
		notifications: make(chan Notification, notifyQBufferSize),
	}
}

func (s *resultStore) gc() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lggr.Println("garbage collecting result store")

	for k, v := range s.data {
		if time.Since(v.addedAt) > storeTTL {
			delete(s.data, k)
			s.notify(NotifyOpEvict, v.data)
		}
	}
}

// notify writes to the notifications channel.
// NOTE: we drop notifications in case the channel is full
func (s *resultStore) notify(op NotifyOp, data ocr2keepers.CheckResult) {
	select {
	case s.notifications <- Notification{
		Op:   op,
		Data: data,
	}:
	default:
		s.lggr.Println("q is full, dropping result")
	}
}

func (s *resultStore) Notifications() <-chan Notification {
	return s.notifications
}

func (s *resultStore) Add(results ...ocr2keepers.CheckResult) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, result := range results {
		id := result.Payload.ID
		el, ok := s.data[id]
		if !ok {
			el = element{}
		}
		// TBD: what if the element is already exists?
		el.data = result
		el.addedAt = time.Now()
		s.data[id] = el
	}
}

func (s *resultStore) Remove(ids ...string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, id := range ids {
		v, ok := s.data[id]
		if !ok {
			continue
		}
		delete(s.data, id)
		s.notify(NotifyOpRemove, v.data)
	}
}

func (s *resultStore) View(filters ...Filter) ([]ocr2keepers.CheckResult, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var result []ocr2keepers.CheckResult
resultLoop:
	for _, r := range s.data {
		for _, filter := range filters {
			if !filter(r.data) {
				continue resultLoop
			}
		}
		result = append(result, r.data)
	}

	return result, nil
}
