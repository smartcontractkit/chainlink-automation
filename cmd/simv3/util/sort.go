package util

import (
	"sort"
	"sync"
)

type SortedKeyMap[T any] struct {
	mu     sync.RWMutex
	values map[string]T
	keys   []string
}

func NewSortedKeyMap[T any]() *SortedKeyMap[T] {
	return &SortedKeyMap[T]{
		values: make(map[string]T),
		keys:   make([]string, 0),
	}
}

func (m *SortedKeyMap[T]) Set(key string, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.values[key]
	if !ok {
		m.keys = append(m.keys, key)
		sort.Strings(m.keys)
	}

	m.values[key] = value
}

func (m *SortedKeyMap[T]) Get(key string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	v, ok := m.values[key]
	if ok {
		return v, ok
	}

	return getZero[T](), false
}

// Keys returns the specified number of keys sorted highest to lowest.
func (m *SortedKeyMap[T]) Keys(count int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keysLen := len(m.keys)

	if count > keysLen {
		count = keysLen
	}

	// only return the last 'count' keys
	keys := make([]string, count)

	// keys are sorted internally in ascending order but the return
	// should be decending
	// loop starting at 1 so the first insert can be l-1, or the last item
	for i := 1; i <= count; i++ {
		keys[i-1] = m.keys[keysLen-i]
	}

	return keys
}

func getZero[T any]() T {
	var result T
	return result
}
