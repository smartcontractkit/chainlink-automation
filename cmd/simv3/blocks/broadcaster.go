package blocks

import (
	"log"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
)

type BlockLoader interface {
	Load(*config.SymBlock)
}

type BlockBroadcaster struct {
	nextBlock     *big.Int
	cadence       time.Duration
	maxDelay      int
	limit         *big.Int
	start         sync.Once
	mu            sync.Mutex
	done          chan struct{}
	subscriptions map[int]chan config.SymBlock
	delays        map[int]bool
	loaders       []BlockLoader
	jitter        time.Duration
	subCount      int
	activeSubs    int
}

func NewBlockBroadcaster(conf config.Blocks, maxDelay int, loaders ...BlockLoader) *BlockBroadcaster {
	limit := new(big.Int).Add(conf.Genesis, big.NewInt(int64(conf.Duration)))

	// add a block padding to allow all transmits to come through
	limit = new(big.Int).Add(limit, big.NewInt(int64(conf.EndPadding)))

	return &BlockBroadcaster{
		nextBlock:     conf.Genesis,
		cadence:       conf.Cadence.Value(),
		maxDelay:      maxDelay,
		loaders:       loaders,
		limit:         limit,
		jitter:        conf.Jitter.Value(),
		done:          make(chan struct{}),
		subscriptions: make(map[int]chan config.SymBlock),
		delays:        make(map[int]bool),
	}
}

func (bb *BlockBroadcaster) run() {
	timer := time.NewTimer(bb.cadenceWithJitter())

	// broadcast the first block immediately
	bb.broadcast()

	for {
		select {
		case <-timer.C:
			bb.mu.Lock()
			bb.nextBlock = new(big.Int).Add(bb.nextBlock, big.NewInt(1))
			bb.mu.Unlock()
			log.Printf("next block: %s", bb.nextBlock)

			if bb.nextBlock.Cmp(bb.limit) > 0 {
				bb.done <- struct{}{}
			} else {
				bb.broadcast()
			}
			timer.Reset(bb.cadenceWithJitter())
		case <-bb.done:
			timer.Stop()
			return
		}
	}
}

func (bb *BlockBroadcaster) cadenceWithJitter() time.Duration {
	if bb.jitter > 0 {
		jitter := rand.Intn(int(bb.jitter))
		half := float64(bb.jitter) / 2
		applied := math.Round(float64(jitter) - half)

		// plus or minus jitter amount
		return bb.cadence + time.Duration(int64(applied))
	}

	return bb.cadence
}

func (bb *BlockBroadcaster) broadcast() {
	msg := config.SymBlock{
		BlockNumber: new(big.Int),
	}

	*msg.BlockNumber = *bb.nextBlock

	for _, loader := range bb.loaders {
		loader.Load(&msg)
	}

	for sub, chSub := range bb.subscriptions {
		go func(ch chan config.SymBlock, delay bool) {
			defer func() {
				if err := recover(); err != nil {
					log.Println(err)
				}
			}()

			if delay {
				// add up to a 2 second delay at random
				r := rand.Intn(bb.maxDelay)
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
	bb.activeSubs++
	bb.subscriptions[bb.subCount] = make(chan config.SymBlock)
	bb.delays[bb.subCount] = delay
	return bb.subCount, bb.subscriptions[bb.subCount]
}

func (bb *BlockBroadcaster) Unsubscribe(subscriptionId int) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	sub, ok := bb.subscriptions[subscriptionId]
	if ok {
		bb.activeSubs--
		close(sub)
	}
	delete(bb.subscriptions, subscriptionId)
	delete(bb.delays, subscriptionId)
}

func (bb *BlockBroadcaster) Start() chan struct{} {
	bb.start.Do(func() {
		go bb.run()
	})
	return bb.done
}

func (bb *BlockBroadcaster) Stop() {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	close(bb.done)
}
