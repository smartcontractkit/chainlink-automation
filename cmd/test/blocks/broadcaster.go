package blocks

import (
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
)

type BlockBroadcaster struct {
	nextBlock     *big.Int
	cadence       time.Duration
	start         sync.Once
	mu            sync.Mutex
	done          chan struct{}
	subscriptions map[int]chan config.SymBlock
	delays        map[int]bool
	subCount      int
}

func NewBlockBroadcaster(conf config.Blocks) *BlockBroadcaster {
	return &BlockBroadcaster{
		nextBlock:     conf.Genesis,
		cadence:       conf.Cadence,
		done:          make(chan struct{}),
		subscriptions: make(map[int]chan config.SymBlock),
	}
}

func (bb *BlockBroadcaster) run() {
	t := time.NewTicker(bb.cadence)
	bb.broadcast()

	for {
		select {
		case <-t.C:
			bb.mu.Lock()
			bb.nextBlock = bb.nextBlock.Add(bb.nextBlock, big.NewInt(1))
			bb.mu.Unlock()
			bb.broadcast()
		case <-bb.done:
			return
		}
	}
}

func (bb *BlockBroadcaster) broadcast() {
	for sub, chSub := range bb.subscriptions {
		msg := config.SymBlock{
			BlockNumber: bb.nextBlock,
		}

		go func(ch chan config.SymBlock, delay bool) {
			if delay {
				r := rand.Intn(5000)
				<-time.After(time.Duration(int64(r)) * time.Millisecond)
			}
			ch <- msg
		}(chSub, bb.delays[sub])
	}
}

func (bb *BlockBroadcaster) Subscribe(delay bool) (int, chan config.SymBlock) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	bb.subCount++
	bb.subscriptions[bb.subCount] = make(chan config.SymBlock)
	bb.delays[bb.subCount] = delay
	return bb.subCount, bb.subscriptions[bb.subCount]
}

func (bb *BlockBroadcaster) Start() {
	bb.start.Do(func() {
		go bb.run()
	})
}

func (bb *BlockBroadcaster) Stop() {
	bb.mu.Lock()
	bb.mu.Unlock()

	close(bb.done)
}
