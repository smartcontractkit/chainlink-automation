package chain

import (
	"log"
	"runtime"
	"sync"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/util"
)

const (
	defaultBlockHistoryChannelDepth = 100
	defaultHistoryDepth             = 256
)

type BlockHistoryTracker struct {
	// provided dependencies
	listener *Listener
	logger   *log.Logger

	// internal state values
	mu       sync.RWMutex
	channels map[int]chan ocr2keepers.BlockHistory
	history  *util.SortedKeyMap[Block]
	count    int

	// service values
	chDone chan struct{}
}

func NewBlockHistoryTracker(listener *Listener, logger *log.Logger) *BlockHistoryTracker {
	tracker := &BlockHistoryTracker{
		listener: listener,
		logger:   logger,
		channels: make(map[int]chan ocr2keepers.BlockHistory),
		history:  util.NewSortedKeyMap[Block](),
		chDone:   make(chan struct{}),
	}

	go tracker.run()

	runtime.SetFinalizer(tracker, func(srv *BlockHistoryTracker) { srv.stop() })

	return tracker
}

// Subscribe provides an identifier integer, a new channel, and potentially an error
func (ht *BlockHistoryTracker) Subscribe() (int, chan ocr2keepers.BlockHistory, error) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	chHistory := make(chan ocr2keepers.BlockHistory, defaultBlockHistoryChannelDepth)
	ht.count++

	ht.channels[ht.count] = chHistory

	return ht.count, chHistory, nil
}

// Unsubscribe requires an identifier integer and indicates the provided channel should be closed
func (ht *BlockHistoryTracker) Unsubscribe(channelID int) error {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if chOpen, ok := ht.channels[channelID]; ok {
		close(chOpen)
		delete(ht.channels, channelID)
	}

	return nil
}

func (ht *BlockHistoryTracker) run() {
	chEvents := ht.listener.Subscribe(BlockChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case Block:
				ht.history.Set(evt.Number.String(), evt)
				ht.broadcast()
			}
		case <-ht.chDone:
			return
		}
	}
}

func (ht *BlockHistoryTracker) stop() {
	close(ht.chDone)
}

func (ht *BlockHistoryTracker) broadcast() {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	history := []ocr2keepers.BlockKey{}

	keys := ht.history.Keys(defaultHistoryDepth)
	for _, key := range keys {
		block, _ := ht.history.Get(key)

		history = append(history, ocr2keepers.BlockKey{
			Number: ocr2keepers.BlockNumber(block.Number.Uint64()),
			Hash:   block.Hash,
		})
	}

	for _, chOpen := range ht.channels {
		chOpen <- history
	}
}
