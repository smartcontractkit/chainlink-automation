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

// Ticker is a process that runs interval ticks.
type Ticker interface {
	Start() error
	Stop() error
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
	ctx      context.Context
	cancelFn context.CancelFunc
}

func NewTimeTicker(interval time.Duration, observer observer, getterFn getUpkeepsFn) timeTicker {
	ctx, cancelFn := context.WithCancel(context.Background())
	t := timeTicker{
		interval: interval,
		ticker:   time.NewTicker(interval),
		observer: observer,
		getterFn: getterFn,
		ctx:      ctx,
		cancelFn: cancelFn,
	}

	return t
}

func (t timeTicker) Start() {
	for tm := range t.ticker.C {
		func() {
			ctx, cancelFn := context.WithTimeout(t.ctx, t.interval)
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
}

func (t timeTicker) Stop() {
	t.cancelFn()
	t.ticker.Stop()
}