package tickers

import (
	"context"
	"log"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

type basicTicker[T any] struct {
	timeTicker[[]T]
	next   *util.Cache[T]
	logger *log.Logger
}

func NewBasicTicker[T any](
	interval time.Duration,
	observer observer[[]T],
	logger *log.Logger,
) *basicTicker[T] {
	bt := &basicTicker[T]{
		next:   util.NewCache[T](time.Hour), // TODO: maybe make this configurable
		logger: logger,
	}

	bt.timeTicker = *NewTimeTicker[[]T](interval, observer, bt.getterFn, logger)

	return bt
}

func (bt *basicTicker[T]) Add(key string, value T) error {
	bt.next.Set(key, value, util.DefaultCacheExpiration)

	return nil
}

func (bt *basicTicker[T]) getterFn(_ context.Context, t time.Time) (Tick[[]T], error) {
	bt.next.ClearExpired()

	allValues := []T{}

	for _, key := range bt.next.Keys() {
		value, ok := bt.next.Get(key)
		if !ok {
			continue
		}

		allValues = append(allValues, value)

		bt.next.Delete(key)
	}

	return staticTick[[]T]{
		value: allValues,
	}, nil
}

type staticTick[V any] struct {
	value V
}

// Values returns the values contained in the Tick.
func (t staticTick[V]) Value(ctx context.Context) (V, error) {
	return t.value, nil
}
