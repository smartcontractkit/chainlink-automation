package instructions

import "sync"

type InstructionStore struct {
	mu     sync.RWMutex
	values map[Instruction]bool
}

func NewStore() *InstructionStore {
	return &InstructionStore{
		values: make(map[Instruction]bool),
	}
}

func (store *InstructionStore) Set(key Instruction) {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.values[key] = true
}

func (store *InstructionStore) Delete(key Instruction) {
	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.values, key)
}

func (store *InstructionStore) Has(key Instruction) bool {
	store.mu.RLock()
	defer store.mu.RUnlock()

	val, ok := store.values[key]
	if !ok {
		return false
	}

	return val
}
