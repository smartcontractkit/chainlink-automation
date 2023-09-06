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

func (m *SortedKeyMap[T]) Keys(l int) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if l > len(m.keys) {
		l = len(m.keys)
	}

	// keys are sorted ascending by block number
	// only return the last 'l' keys
	keys := make([]string, l)
	// loop starting at 1 so the first insert can be l-1, or the last item
	for i := 1; i <= l; i++ {
		keys[i-1] = m.keys[l-i]
	}

	return keys
}

func getZero[T any]() T {
	var result T
	return result
}
