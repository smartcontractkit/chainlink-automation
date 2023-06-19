package tickers

import (
	"context"
	"fmt"
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var (
	ErrNotRetryable          = fmt.Errorf("payload is not retryable")
	ErrRetryDurationExceeded = fmt.Errorf("payload has exceed allowed retry window")
)

const (
	DefaultRetryDelay           = 10 * time.Second
	DefaultMaxRetryDuration     = 5 * time.Minute
	DefaultRetryCacheExpiration = 2 * DefaultMaxRetryDuration // 10 minutes
)

type RetryConfig struct {
	RetryDelay           time.Duration
	MaxRetryDuration     time.Duration
	RetryCacheExpiration time.Duration
}

type RetryConfigFunc func(*RetryConfig)

var RetryWithDefaults = func(c *RetryConfig) {
	c.RetryDelay = DefaultRetryDelay
	c.MaxRetryDuration = DefaultMaxRetryDuration
	c.RetryCacheExpiration = DefaultRetryCacheExpiration
}

type retryTicker struct {
	timeTicker
	config         RetryConfig
	nextRetries    sync.Map // time.Time -> ocr2keepers.UpkeepPayload
	retryEntries   *util.Cache[time.Time]
	payloadModFunc func(ocr2keepers.UpkeepPayload) ocr2keepers.UpkeepPayload
}

// Retry adds a retryable result to the retryTicker.
func (rt *retryTicker) Retry(result ocr2keepers.CheckResult) error {
	payload := result.Payload

	if !result.Retryable {
		// exit condition for not retryable
		return fmt.Errorf("%w: %s", ErrNotRetryable, payload.ID)
	}

	entryTime, ok := rt.retryEntries.Get(payload.ID)
	if !ok {
		entryTime = time.Now()
		rt.retryEntries.Set(payload.ID, entryTime, util.DefaultCacheExpiration)
	}

	now := time.Now()

	if now.Sub(entryTime) > rt.config.MaxRetryDuration {
		// exit condition for exceeding maximum retry time
		return fmt.Errorf("%w: %s", ErrRetryDurationExceeded, payload.ID)
	}

	rt.nextRetries.Store(now.Add(rt.config.RetryDelay), payload)

	return nil
}

// getterFn is a function that retrieves the retryTick for the given time.
func (rt *retryTicker) getterFn(ctx context.Context, t time.Time) (Tick, error) {
	rt.retryEntries.ClearExpired()

	upkeepPayloads := []ocr2keepers.UpkeepPayload{}

	rt.nextRetries.Range(func(key, value interface{}) bool {
		if runTime, ok := key.(time.Time); ok {
			if t.After(runTime) {
				if payload, ok := value.(ocr2keepers.UpkeepPayload); ok {
					upkeepPayloads = append(upkeepPayloads, rt.payloadModFunc(payload))
				}
				rt.nextRetries.Delete(key)
			}
		}
		return true
	})

	return retryTick{
		upkeeps: upkeepPayloads,
	}, nil
}

// NewRetryTicker creates a new retryTicker with the specified interval and observer.
func NewRetryTicker(interval time.Duration, observer observer, configFuncs ...RetryConfigFunc) *retryTicker {
	config := RetryConfig{}

	if len(configFuncs) == 0 {
		RetryWithDefaults(&config)
	} else {
		for _, f := range configFuncs {
			f(&config)
		}
	}

	rt := &retryTicker{
		config:       config,
		nextRetries:  sync.Map{},
		retryEntries: util.NewCache[time.Time](config.RetryCacheExpiration),
		payloadModFunc: func(p ocr2keepers.UpkeepPayload) ocr2keepers.UpkeepPayload {
			return p
		},
	}

	rt.timeTicker = *NewTimeTicker(interval, observer, rt.getterFn)

	return rt
}

type retryTick struct {
	upkeeps []ocr2keepers.UpkeepPayload
}

// GetUpkeeps returns the upkeeps contained in the retryTick.
func (t retryTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return t.upkeeps, nil
}
