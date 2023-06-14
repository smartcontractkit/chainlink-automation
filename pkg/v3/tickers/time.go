package tickers

import (
	"context"
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

var (
	logPrintf = log.Printf
)

type observer interface {
	Process(context.Context, Tick) error
}

// Tick is the container for the individual tick
type Tick interface {
	// GetUpkeeps provides upkeeps scoped to the tick
	GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error)
}

type getUpkeepsFn func(context.Context, time.Time) (Tick, error)

type timeTicker struct {
	interval time.Duration
	ticker   *time.Ticker
	observer observer
	getterFn getUpkeepsFn
	chClose  chan struct{}
}

func NewTimeTicker(interval time.Duration, observer observer, getterFn getUpkeepsFn) *timeTicker {
	t := &timeTicker{
		interval: interval,
		ticker:   time.NewTicker(interval),
		observer: observer,
		getterFn: getterFn,
		chClose:  make(chan struct{}, 1),
	}

	return t
}

// Start uses the provided context for each call to the getter function with the
// configured interval as a timeout. This function blocks until Close is called
// or the parent context is cancelled.
func (t *timeTicker) Start(ctx context.Context) error {
	for tm := range t.ticker.C {
		ctx, cancelFn := context.WithTimeout(ctx, t.interval)

		go func() {
			select {
			case <-ctx.Done():
				cancelFn()
			case <-t.chClose:
				cancelFn()
			}
		}()

		func() {
			defer cancelFn()

			tick, err := t.getterFn(ctx, tm)
			if err != nil {
				logPrintf("error fetching tick: %s", err.Error())
			}

			if err := t.observer.Process(ctx, tick); err != nil {
				logPrintf("error processing observer: %s", err.Error())
			}
		}()
	}

	return nil
}

func (t *timeTicker) Close() error {
	t.ticker.Stop()
	t.chClose <- struct{}{}

	return nil
}
