package util

import (
	"sync"
)

// Queue saves generic data in buckets, each bucket has a key and it holds a list of items.
type Queue[V any] struct {
	data []V
	mu   *sync.RWMutex
}

// NewQueue creates a new queue with the given generic types
func NewQueue[V any]() *Queue[V] {
	return &Queue[V]{
		data: make([]V, 0),
		mu:   &sync.RWMutex{},
	}
}

// Push adds items to the q, it is possible to add values of multiple buckets
func (q *Queue[V]) Push(vals ...V) {
	if len(vals) == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	q.data = append(q.data, vals...)
}

// Pop returns the corresponding items and removed them from the q.
// if n is <= 0 then all the items will be popped.
func (q *Queue[V]) Pop(n int) []V {
	q.mu.Lock()
	defer q.mu.Unlock()

	size := len(q.data)
	if size == 0 {
		return nil
	}
	// ensure we don't overflow
	if n > size || n <= 0 {
		n = size
	}
	removed, newlist := q.data[:n], q.data[n:]
	q.data = newlist

	return removed
}

// PopF accept a filter function to determine which items to pop
func (q *Queue[V]) PopF(filter func(V) bool) []V {
	q.mu.Lock()
	defer q.mu.Unlock()

	removed := make([]V, 0)
	updated := make([]V, 0)
	for i, v := range q.data {
		if filter(v) {
			removed = append(removed, q.data[i])
		} else {
			updated = append(updated, v)
		}
	}

	q.data = updated

	return removed
}

// Size returs the size of the list for a specific key
func (q *Queue[V]) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.data)
}
