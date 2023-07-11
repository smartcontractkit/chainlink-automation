package tickers

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
}

type shuffler[T any] interface {
	Shuffle([]T) []T
}

type upkeepsGetter interface {
	GetActiveUpkeeps(context.Context, ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error)
}

type sampleTicker struct {
	// provided dependencies
	observer      observer
	getter        upkeepsGetter
	ratio         ratio
	samplingLimit time.Duration
	logger        *log.Logger

	// set by constructor
	blocks   *BlockTicker
	shuffler shuffler[ocr2keepers.UpkeepPayload]

	// run state
	running atomic.Bool
	chClose chan struct{}
}

func (st *sampleTicker) Start(ctx context.Context) error {
	if st.running.Load() {
		return fmt.Errorf("already running")
	}

	go func() {
		if err := st.blocks.Start(ctx); err != nil {
			st.logger.Printf("error starting block ticker: %s", err)
		}
	}()

	st.running.Store(true)

	ctx, cancel := context.WithCancel(ctx)

Loop:
	for {
		select {
		case h := <-st.blocks.C:
			latestBlock, err := h.Latest()
			if err != nil {
				continue Loop
			}

			// do the observation with limited time
			ctx, cancelFn := context.WithTimeout(ctx, st.samplingLimit)

			tick, err := st.getterFn(ctx, latestBlock)
			if err != nil {
				st.logger.Printf("failed to get upkeeps: %s", err)
				cancelFn()

				continue Loop
			}

			if err := st.observer.Process(ctx, tick); err != nil {
				st.logger.Printf("error processing observer: %s", err)
			}

			cancelFn()
		case <-st.chClose:
			break Loop
		}
	}

	cancel()

	return nil
}

func (st *sampleTicker) Close() error {
	if !st.running.Load() {
		return fmt.Errorf("not running")
	}

	st.blocks.Close()
	st.chClose <- struct{}{}

	return nil
}

func (ticker *sampleTicker) getterFn(ctx context.Context, block ocr2keepers.BlockKey) (Tick, error) {
	var (
		upkeeps []ocr2keepers.UpkeepPayload
		err     error
	)

	// TODO: convert to block key ticker instead of time ticker to provide
	// block scope to active upkeep provider
	if upkeeps, err = ticker.getter.GetActiveUpkeeps(ctx, block); err != nil {
		return nil, err
	}

	upkeeps = ticker.shuffler.Shuffle(upkeeps)
	size := ticker.ratio.OfInt(len(upkeeps))

	if len(upkeeps) == 0 || size <= 0 {
		return upkeepTick{upkeeps: nil}, nil
	}

	if len(upkeeps) <= size {
		return upkeepTick{upkeeps: upkeeps}, nil
	}

	return upkeepTick{upkeeps: upkeeps[:size]}, nil
}

func NewSampleTicker(
	ratio ratio,
	getter upkeepsGetter,
	observer observer,
	subscriber blockSubscriber,
	samplingLimit time.Duration,
	logger *log.Logger,
) (*sampleTicker, error) {
	block, err := NewBlockTicker(subscriber)
	if err != nil {
		return nil, err
	}

	st := &sampleTicker{
		observer:      observer,
		getter:        getter,
		ratio:         ratio,
		samplingLimit: samplingLimit,
		logger:        logger,
		blocks:        block,
		shuffler:      util.Shuffler[ocr2keepers.UpkeepPayload]{Source: util.NewCryptoRandSource()},
		chClose:       make(chan struct{}, 1),
	}

	return st, nil
}
