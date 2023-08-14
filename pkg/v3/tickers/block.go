package tickers

import (
	"context"
	"log"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// BlockTicker is a struct that follows the same design paradigm as a time ticker but provides blocks
// instead of time
type BlockTicker struct {
	closer util.Closer

	C             chan ocr2keepers.BlockHistory
	chID          int
	ch            chan ocr2keepers.BlockHistory
	subscriber    ocr2keepers.BlockSubscriber
	bufferedValue ocr2keepers.BlockHistory
	nextCh        chan ocr2keepers.BlockHistory
}

func NewBlockTicker(subscriber ocr2keepers.BlockSubscriber) (*BlockTicker, error) {
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
	}, nil
}

func (t *BlockTicker) Start(pctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	if !t.closer.Store(cancel) {
		return nil
	}

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
}

func (t *BlockTicker) Close() error {
	if !t.closer.Close() {
		return nil
	}

	if err := t.subscriber.Unsubscribe(t.chID); err != nil {
		log.Printf("error unsubscribing: %v", err)
	}

	return nil
}
