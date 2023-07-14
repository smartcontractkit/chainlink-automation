package tickers

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

type observer[T any] interface {
	Process(context.Context, Tick[T]) error
}

type getterFunc[T any] func(context.Context, time.Time) (Tick[T], error)

type timeTicker[T any] struct {
	interval time.Duration
	ticker   *time.Ticker
	observer observer[T]
	getterFn getterFunc[T]
	logger   *log.Logger
	chClose  chan struct{}
	running  atomic.Bool
}

func NewTimeTicker[T any](interval time.Duration, observer observer[T], getterFn getterFunc[T], logger *log.Logger) *timeTicker[T] {
	t := &timeTicker[T]{
		interval: interval,
		ticker:   time.NewTicker(interval),
		observer: observer,
		getterFn: getterFn,
		logger:   logger,
		chClose:  make(chan struct{}, 1),
	}

	return t
}

// Start uses the provided context for each call to the getter function with the
// configured interval as a timeout. This function blocks until Close is called
// or the parent context is cancelled.
func (t *timeTicker[T]) Start(ctx context.Context) error {
	if t.running.Load() {
		return fmt.Errorf("already running")
	}

	t.running.Store(true)

	ctx, cancel := context.WithCancel(ctx)

Loop:
	for {
		select {
		case tm := <-t.ticker.C:
			tick, err := t.getterFn(ctx, tm)
			if err != nil {
				t.logger.Printf("error fetching tick: %s", err.Error())
			}

			if err := t.observer.Process(ctx, tick); err != nil {
				t.logger.Printf("error processing observer: %s", err.Error())
			}
		case <-t.chClose:
			break Loop
		}
	}

	cancel()

	t.running.Store(false)

	return nil
}

func (t *timeTicker[T]) Close() error {
	if !t.running.Load() {
		return fmt.Errorf("not running")
	}

	t.ticker.Stop()
	t.chClose <- struct{}{}

	return nil
}
