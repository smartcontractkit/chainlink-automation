package blocks

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
)

type BlockBroadcaster struct {
	nextBlock        *big.Int
	cadence          time.Duration
	limit            *big.Int
	start            sync.Once
	mu               sync.Mutex
	done             chan struct{}
	subscriptions    map[int]chan config.SymBlock
	delays           map[int]bool
	transmits        map[string][]byte
	transmitEpochs   map[string]uint32
	transmitInBlock  map[string]*big.Int
	nextTransmitHash *string
	subCount         int
	activeSubs       int
}

func NewBlockBroadcaster(conf config.Blocks) *BlockBroadcaster {
	return &BlockBroadcaster{
		nextBlock:     conf.Genesis,
		cadence:       conf.Cadence,
		limit:         big.NewInt(0).Add(conf.Genesis, big.NewInt(int64(conf.Duration))),
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
			bb.nextBlock = big.NewInt(0).Add(bb.nextBlock, big.NewInt(1))
			bb.mu.Unlock()

			if bb.nextBlock.Cmp(bb.limit) > 0 {
				bb.done <- struct{}{}
			} else {
				bb.broadcast()
			}
		case <-bb.done:
			return
		}
	}
}

func (bb *BlockBroadcaster) broadcast() {
	for sub, chSub := range bb.subscriptions {
		msg := config.SymBlock{}

		*msg.BlockNumber = *bb.nextBlock

		if bb.nextTransmitHash != nil {
			bb.nextTransmitHash = nil
			report, ok := bb.transmits[*bb.nextTransmitHash]
			if ok {
				msg.TransmittedData = report
			}

			epoch, ok := bb.transmitEpochs[*bb.nextTransmitHash]
			if ok {
				msg.LatestEpoch = &epoch
			}
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

func (bb *BlockBroadcaster) Transmit(report []byte, epoch uint32) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	reportHash := hashReport(report)

	if e, ok := bb.transmitEpochs[reportHash]; ok && e == epoch {
		return fmt.Errorf("report already transmitted in epoch %d", epoch)
	}

	bb.transmitEpochs[reportHash] = epoch
	bb.transmits[reportHash] = report
	bb.nextTransmitHash = &reportHash
	*bb.transmitInBlock[reportHash] = *bb.nextBlock

	return nil
}

func (bb *BlockBroadcaster) Start() {
	bb.start.Do(func() {
		go bb.run()
	})
}

func (bb *BlockBroadcaster) Stop() {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	close(bb.done)
}

func hashReport(report []byte) string {
	hasher := sha256.New()
	hasher.Write(report)
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
