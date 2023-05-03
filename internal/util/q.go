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
	q.mu.Lock()
	defer q.mu.Unlock()

	q.data = append(q.data, vals...)
}

// Pop returns the corresponding items and removed them from the q
// TBD: amount of items to pop vs. filter function
func (q *Queue[V]) Pop(n int) []V {
	q.mu.Lock()
	defer q.mu.Unlock()

	newlist, removed := pop(q.data, n)
	q.data = newlist

	return removed
}

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

// Pop returns the corresponding items and removed them from the q
func pop[V any](list []V, n int) ([]V, []V) {
	size := len(list)
	// ensure we don't overflow by auto-correct n
	if n > size || n <= 0 {
		n = size
	}
	removed := list[:n]
	list = list[n:]

	return list, removed
}
