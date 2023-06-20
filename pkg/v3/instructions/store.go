package instructions

import "sync"

type instructionStore struct {
	mu     sync.RWMutex
	values map[Instruction]bool
}

func NewStore() *instructionStore {
	return &instructionStore{
		values: make(map[Instruction]bool),
	}
}

func (store *instructionStore) Set(key Instruction) {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.values[key] = true
}

func (store *instructionStore) Delete(key Instruction) {
	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.values, key)
}

func (store *instructionStore) Has(key Instruction) bool {
	store.mu.RLock()
	defer store.mu.RUnlock()

	val, ok := store.values[key]
	if !ok {
		return false
	}

	return val
}
