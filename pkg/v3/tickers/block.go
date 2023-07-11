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

// BlockTicker is a struct that follows the same design paradigm as a time ticker but provides blocks
// instead of time
type BlockTicker struct {
	C             chan ocr2keepers.BlockHistory
	chID          int
	ch            chan ocr2keepers.BlockHistory
	subscriber    blockSubscriber
	bufferedValue ocr2keepers.BlockHistory
	nextCh        chan ocr2keepers.BlockHistory
	closer        sync.Once
	stopCh        chan int
}

func NewBlockTicker(subscriber blockSubscriber) (*BlockTicker, error) {
	chID, ch, err := subscriber.Subscribe()
	if err != nil {
		return nil, err
	}

	return &BlockTicker{
		chID:       chID,
		ch:         ch,
		C:          make(chan ocr2keepers.BlockHistory),
		nextCh:     make(chan ocr2keepers.BlockHistory),
		subscriber: subscriber,
		closer:     sync.Once{},
		stopCh:     make(chan int),
	}, nil
}

func (t *BlockTicker) Start(ctx context.Context) (err error) {
loop:
	for {
		select {
		case blockHistory := <-t.ch:
			select {
			case t.C <- blockHistory:
				t.bufferedValue = nil
			default:
				t.bufferedValue = blockHistory
			}
		case <-ctx.Done():
			err = ctx.Err()
			break loop
		case <-t.stopCh:
			return nil
		default:
			if t.bufferedValue != nil {
				select {
				case t.C <- t.bufferedValue:
					t.bufferedValue = nil
				default:
				}
			}
		}
	}
	return err
}

func (t *BlockTicker) Close() {
	t.closer.Do(func() {
		t.stopCh <- 1
		if err := t.subscriber.Unsubscribe(t.chID); err != nil {
			log.Printf("error unsubscribing: %v", err)
		}
	})
}
