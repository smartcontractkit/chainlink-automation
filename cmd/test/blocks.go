package main

import (
	"sync"
	"time"
)

type BlockSynchronizer struct {
	GenesisBlock uint64
	BlockTime    time.Duration
	WithJitter   bool
	chBlock      chan uint64
	start        sync.Once
	stop         sync.Once
}

func (bs *BlockSynchronizer) Start() {
	bs.start.Do(func() {

	})
}

func (bs *BlockSynchronizer) Stop() {
	bs.stop.Do(func() {

	})
}
