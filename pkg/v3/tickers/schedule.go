/*
ScheduleTicker produces includes an added value in a tick SendDelay duration
after the value is scheduled.
*/
package tickers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var (
	ErrSendDurationExceeded = fmt.Errorf("scheduled value has exceed allowed send window")
)

const (
	DefaultSendDelay       = 10 * time.Second
	DefaultMaxSendDuration = 5 * time.Minute
)

type ScheduleTickerConfig struct {
	// SendDelay is the time to wait before re-sending the item
	SendDelay time.Duration
	// MaxSendDuration is the total amount of time to attempt sending
	MaxSendDuration time.Duration
}

type ScheduleTickerConfigFunc func(*ScheduleTickerConfig)

var ScheduleTickerWithDefaults = func(c *ScheduleTickerConfig) {
	c.SendDelay = DefaultSendDelay
	c.MaxSendDuration = DefaultMaxSendDuration
}

type scheduleTicker[T any] struct {
	timeTicker[[]T]
	config          ScheduleTickerConfig
	scheduled       *util.Cache[T]
	scheduledLookup *util.Cache[time.Time]
	importer        func(func(key string, value T) error) error
	logger          *log.Logger
}

func NewScheduleTicker[T any](
	interval time.Duration,
	observer observer[[]T],
	importer func(func(key string, value T) error) error,
	logger *log.Logger,
	configFuncs ...ScheduleTickerConfigFunc,
) *scheduleTicker[T] {
	config := ScheduleTickerConfig{}

	if len(configFuncs) == 0 {
		ScheduleTickerWithDefaults(&config)
	} else {
		for _, f := range configFuncs {
			f(&config)
		}
	}

	st := &scheduleTicker[T]{
		config:          config,
		scheduled:       util.NewCache[T](5 * config.MaxSendDuration),
		scheduledLookup: util.NewCache[time.Time](5 * config.MaxSendDuration),
		importer:        importer,
		logger:          logger,
	}

	st.timeTicker = *NewTimeTicker[[]T](interval, observer, st.getterFn, logger)

	return st
}

func (st *scheduleTicker[T]) Schedule(key string, value T) error {
	now := time.Now()

	entryTime, ok := st.scheduledLookup.Get(key)
	if !ok {
		entryTime = now
		st.scheduledLookup.Set(key, entryTime, util.DefaultCacheExpiration)
	}

	if now.Sub(entryTime) > st.config.MaxSendDuration {
		// exit condition for exceeding maximum retry time
		return fmt.Errorf("%w: %s", ErrSendDurationExceeded, key)
	}

	st.scheduled.Set(fmt.Sprintf("%d", now.Add(st.config.SendDelay).UnixNano()), value, util.DefaultCacheExpiration)

	return nil
}

func (st *scheduleTicker[T]) getterFn(_ context.Context, t time.Time) (Tick[[]T], error) {
	st.scheduled.ClearExpired()
	st.scheduledLookup.ClearExpired()

	// the importer pull values from an outside source into the schedule
	if err := st.importer(st.Schedule); err != nil {
		return nil, err
	}

	allValues := []T{}

	for _, nanoStr := range st.scheduled.Keys() {
		// nanoStr is UnixNano stringified
		nano, err := strconv.ParseInt(nanoStr, 10, 64)
		if err != nil {
			st.logger.Printf("schedule ticker skipped value for key %s because key was not int64", nanoStr)
			continue
		}

		runTime := time.Unix(0, nano)
		if t.After(runTime) {
			if value, ok := st.scheduled.Get(nanoStr); ok {
				allValues = append(allValues, value)
			}

			st.scheduled.Delete(nanoStr)
		}
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
