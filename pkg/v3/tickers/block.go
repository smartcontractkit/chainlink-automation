package tickers

import (
	"context"
	"log"
	"sync"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type blockSubscriber interface {
	// Subscribe provides an identifier integer, a new channel, and potentially an error
	Subscribe() (int, chan ocr2keepers.BlockHistory, error)
	// Unsubscribe requires an identifier integer and indicates the provided channel should be closed
	Unsubscribe(int) error
}

// blockTicker is a struct that follows the same design paradigm as a time ticker but provides blocks
// instead of time
type blockTicker struct {
	C          chan ocr2keepers.BlockHistory
	chID       int
	ch         chan ocr2keepers.BlockHistory
	subscriber blockSubscriber
	closer     sync.Once
	stopCh     chan int
}

func NewBlockTicker(subscriber blockSubscriber) (*blockTicker, error) {
	chID, ch, err := subscriber.Subscribe()
	if err != nil {
		return nil, err
	}

	return &blockTicker{
		chID:       chID,
		ch:         ch,
		C:          make(chan ocr2keepers.BlockHistory),
		subscriber: subscriber,
		closer:     sync.Once{},
		stopCh:     make(chan int),
	}, nil
}

func (t *blockTicker) Start(ctx context.Context) (err error) {
loop:
	for {
		select {
		case blockHistory := <-t.ch:
			select {
			case t.C <- blockHistory:
				log.Print("forwarded")
			default:
			}
		case <-ctx.Done():
			err = ctx.Err()
			break loop
		case <-t.stopCh:
			return nil
		}
	}
	return err
}

func (t *blockTicker) Close() {
	t.closer.Do(func() {
		t.stopCh <- 1
		if err := t.subscriber.Unsubscribe(t.chID); err != nil {
			log.Printf("error unsubscribing: %v", err)
		}
	})
}
