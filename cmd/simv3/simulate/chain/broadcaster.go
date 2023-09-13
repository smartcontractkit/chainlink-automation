package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
)

type BlockLoaderFunc func(*Block)

type BlockBroadcaster struct {
	// provided dependencies
	loaders []BlockLoaderFunc
	logger  *log.Logger

	// configuration
	maxDelay int
	limit    *big.Int
	cadence  time.Duration
	jitter   time.Duration

	// internal state
	mu            sync.RWMutex
	nextBlock     *big.Int
	subscriptions map[int]chan Block
	delays        map[int]bool
	subCount      int
	activeSubs    int

	// service state
	start sync.Once
	done  chan struct{}
}

func NewBlockBroadcaster(conf config.Blocks, maxDelay int, logger *log.Logger, loaders ...BlockLoaderFunc) *BlockBroadcaster {
	limit := new(big.Int).Add(conf.Genesis, big.NewInt(int64(conf.Duration)))

	// add a block padding to allow all transmits to come through
	limit = new(big.Int).Add(limit, big.NewInt(int64(conf.EndPadding)))

	return &BlockBroadcaster{
		loaders:       loaders,
		logger:        log.New(logger.Writer(), "[block-broadcaster] ", log.Ldate|log.Ltime|log.Lshortfile),
		maxDelay:      maxDelay,
		limit:         limit,
		cadence:       conf.Cadence.Value(),
		jitter:        conf.Jitter.Value(),
		nextBlock:     conf.Genesis,
		subscriptions: make(map[int]chan Block),
		delays:        make(map[int]bool),
		done:          make(chan struct{}),
	}
}

func (bb *BlockBroadcaster) Subscribe(delay bool) (int, chan Block) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	bb.subCount++
	bb.activeSubs++
	bb.subscriptions[bb.subCount] = make(chan Block)
	bb.delays[bb.subCount] = delay

	return bb.subCount, bb.subscriptions[bb.subCount]
}

func (bb *BlockBroadcaster) Unsubscribe(subscriptionId int) {
	bb.unsubscribe(subscriptionId, true)
}

func (bb *BlockBroadcaster) unsubscribe(subscriptionId int, closeChan bool) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	sub, ok := bb.subscriptions[subscriptionId]
	if ok {
		bb.activeSubs--

		if closeChan {
			close(sub)
		}
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

			bb.logger.Printf("next block: %s", bb.nextBlock)

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
	bb.mu.RLock()
	defer bb.mu.RUnlock()

	msg := Block{
		Number: new(big.Int).Set(bb.nextBlock),
	}

	for _, loader := range bb.loaders {
		loader(&msg)
	}

	// do a hash of total block
	var bts bytes.Buffer
	_ = gob.NewEncoder(&bts).Encode(msg)

	msg.Hash = sha256.Sum256(bts.Bytes())

	for sub, chSub := range bb.subscriptions {
		go func(subID int, ch chan Block, delay bool, logger *log.Logger) {
			defer func() {
				if err := recover(); err != nil {
					logger.Println(err)

					bb.unsubscribe(subID, false)
				}
			}()

			if delay && bb.maxDelay > 0 {
				// add up to `maxDelay` millisecond delay at random
				r := rand.Intn(bb.maxDelay)

				<-time.After(time.Duration(int64(r)) * time.Millisecond)
			}

			ch <- msg
		}(sub, chSub, bb.delays[sub], bb.logger)
	}
}
