package tickers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/chainlink-automation/internal/util"
)

type observer interface {
	Process(context.Context, Tick) error
}

type getterFunc func(context.Context, time.Time) (Tick, error)

type timeTicker struct {
	closer util.Closer

	interval time.Duration
	observer observer
	getterFn getterFunc
	logger   *log.Logger
}

func NewTimeTicker(interval time.Duration, observer observer, getterFn getterFunc, logger *log.Logger) *timeTicker {
	return &timeTicker{
		interval: interval,
		observer: observer,
		getterFn: getterFn,
		logger:   logger,
	}
}

// Start uses the provided context for each call to the getter function with the
// configured interval as a timeout. This function blocks until Close is called
// or the parent context is cancelled.
func (t *timeTicker) Start(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	if !t.closer.Store(cancel) {
		return fmt.Errorf("already running")
	}

	t.logger.Printf("starting ticker service")
	defer t.logger.Printf("ticker service stopped")

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case tm := <-ticker.C:
			if t.getterFn == nil {
				continue
			}
			tick, err := t.getterFn(ctx, tm)
			if err != nil {
				t.logger.Printf("error fetching tick: %s", err.Error())
			}
			// observer.Process can be a heavy call taking upto ObservationProcessLimit seconds
			// so it is run in a separate goroutine to not block further ticks
			// Exploratory: Add some control to limit the number of goroutines spawned
			go func(c context.Context, t Tick, o observer, l *log.Logger) {
				if err := o.Process(c, t); err != nil {
					l.Printf("error processing observer: %s", err.Error())
				}
			}(ctx, tick, t.observer, t.logger)
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *timeTicker) Close() error {
	_ = t.closer.Close()
	return nil
}
