package resultstore

import (
	"log"
	"sync"
	"time"
)

type KeyFunc[T any] func(t T) string

type NotifyOp uint8

const (
	NotifyOpNil NotifyOp = iota
	NotifyOpEvict
	NotifyOpRemove
)

type Notification[T any] struct {
	Op NotifyOp
	T  T
}

type ResultStore[T any] interface {
	Add(...T)
	Remove(...T)
	View() ([]T, error)
}

var (
	notifyQBufferSize = 128
	storeTTL          = time.Minute
)

type element[T any] struct {
	t       T
	addedAt time.Time
}

type resultStore[T any] struct {
	lggr *log.Logger

	data map[string]element[T]
	lock sync.RWMutex

	keyFn KeyFunc[T]

	notifications chan Notification[T]
}

func NewResultStore[T any](lggr *log.Logger, keyFn KeyFunc[T]) *resultStore[T] {
	return &resultStore[T]{
		lggr:          lggr,
		data:          make(map[string]element[T]),
		lock:          sync.RWMutex{},
		keyFn:         keyFn,
		notifications: make(chan Notification[T], notifyQBufferSize),
	}
}

func (s *resultStore[T]) gc() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lggr.Println("garbage collecting result store")

	for k, v := range s.data {
		if time.Since(v.addedAt) > storeTTL {
			delete(s.data, k)
			s.notify(NotifyOpEvict, v.t)
		}
	}
}

func (s *resultStore[T]) notify(op NotifyOp, t T) {
	n := new(Notification[T])
	n.Op = op
	n.T = t
	select {
	case s.notifications <- *n:
	default:
		s.lggr.Println("q full, dropping result")
	}
}

func (s *resultStore[T]) Notifications() <-chan Notification[T] {
	return s.notifications
}

func (s *resultStore[T]) Add(results ...T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, result := range results {
		key := s.keyFn(result)
		el, ok := s.data[key]
		if !ok {
			el = element[T]{}
		}
		// TBD: what if the element is already there?
		el.t = result
		el.addedAt = time.Now()
		s.data[key] = el
	}
}

func (s *resultStore[T]) Remove(results ...T) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, result := range results {
		key := s.keyFn(result)
		v := s.data[key]
		delete(s.data, key)
		s.notify(NotifyOpRemove, v.t)
	}
}

func (s *resultStore[T]) View() ([]T, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var result []T
	for _, r := range s.data {
		result = append(result, r.t)
	}
	return result, nil
}
