package chain

import (
	"log"
	"math/big"
	"runtime"
	"sync"
)

type EventChannel string

const (
	NotifyChannelLength       = 1000
	SubscriptionChannelLength = 100

	BlockChannel         EventChannel = "block"
	LogTriggerChannel    EventChannel = "log_trigger"
	OCR3ConfigChannel    EventChannel = "ocr3_config"
	PerformUpkeepChannel EventChannel = "perform_upkeep"
	CreateUpkeepChannel  EventChannel = "upkeep_created"
)

type ChainEvent struct {
	BlockNumber *big.Int
	BlockHash   []byte
	Event       interface{}
}

type Broadcaster interface {
	Subscribe(bool) (int, chan Block)
	Unsubscribe(int)
}

// Listener follows blocks on a chain and extracts values from it as they
// become available on simulated chain.
type Listener struct {
	// provided dependencies
	blockSource Broadcaster
	logger      *log.Logger

	// internal state values
	mu             sync.RWMutex
	subscriptionID int
	lastBlock      *big.Int
	blocks         map[string]Block
	subscriptions  map[EventChannel][]chan ChainEvent

	// internal service values
	chDone chan struct{}
}

func NewListener(src Broadcaster, logger *log.Logger) *Listener {
	listener := &Listener{
		blockSource:   src,
		logger:        log.New(logger.Writer(), "[chain-listener] ", log.Ldate|log.Ltime|log.Lshortfile),
		blocks:        make(map[string]Block),
		subscriptions: make(map[EventChannel][]chan ChainEvent),
		chDone:        make(chan struct{}),
	}

	go listener.run()

	runtime.SetFinalizer(listener, func(srv *Listener) { srv.stop() })

	return listener
}

func (cl *Listener) Subscribe(channels ...EventChannel) <-chan ChainEvent {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	chNew := make(chan ChainEvent, SubscriptionChannelLength)

	for _, channel := range channels {
		subs, ok := cl.subscriptions[channel]
		if !ok {
			cl.subscriptions[channel] = []chan ChainEvent{chNew}

			return chNew
		}

		cl.subscriptions[channel] = append(subs, chNew)
	}

	return chNew
}

func (cl *Listener) run() {
	subID, chBlocks := cl.blockSource.Subscribe(true)

	cl.mu.Lock()
	cl.subscriptionID = subID
	cl.mu.Unlock()

	for {
		select {
		case block := <-chBlocks:

			cl.logger.Printf("received block %s", block.Number)

			cl.saveBlock(block)

			// always broadcast the block
			cl.broadcastTransaction(BlockChannel, ChainEvent{
				BlockNumber: block.Number,
				BlockHash:   block.Hash,
				Event:       block,
			})

			for _, transaction := range block.Transactions {
				var channelName EventChannel

				switch transaction.(type) {
				case Log:
					channelName = LogTriggerChannel
				case OCR3ConfigTransaction:
					channelName = OCR3ConfigChannel
				case PerformUpkeepTransaction:
					channelName = PerformUpkeepChannel
				case UpkeepCreatedTransaction:
					channelName = CreateUpkeepChannel
				}

				evt := ChainEvent{
					BlockNumber: block.Number,
					BlockHash:   block.Hash,
					Event:       transaction,
				}

				cl.broadcastTransaction(channelName, evt)
			}
		case <-cl.chDone:
			return
		}
	}
}

func (cl *Listener) saveBlock(block Block) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.blocks[string(block.Hash)] = block
	cl.lastBlock = block.Number
}

func (cl *Listener) broadcastTransaction(channel EventChannel, event ChainEvent) {
	if subs, ok := cl.subscriptions[channel]; ok {
		for i := range subs {
			go func(chSub chan ChainEvent) { chSub <- event }(subs[i])
		}
	}
}

func (cl *Listener) stop() {
	cl.mu.RLock()
	subID := cl.subscriptionID
	cl.mu.RUnlock()

	cl.blockSource.Unsubscribe(subID)

	close(cl.chDone)
}
