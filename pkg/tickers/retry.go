package tickers

import (
	"context"
	"fmt"
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

const (
	retryDelay              = 500 * time.Millisecond
	totalAttempt            = 3
	attemptsCacheExpiration = 10 * retryDelay // 5 seconds
)

type retryTicker struct {
	timeTicker
	nextRetries     sync.Map // time.Time -> ocr2keepers.UpkeepPayload
	payloadAttempts *util.Cache[int]
}

// Retry adds a retryable result to the retryTicker.
func (rt *retryTicker) Retry(result ocr2keepers.CheckResult) error {
	payload := result.Payload

	if result.Retryable {
		attemptCount, ok := rt.payloadAttempts.Get(payload.ID)
		if !ok {
			attemptCount = 0
		}

		if attemptCount >= totalAttempt {
			fmt.Printf("Payload %s has already been tried %d times. Skipping.\n", payload.ID, totalAttempt)
			return nil
		}

		attemptCount++
		rt.payloadAttempts.Set(payload.ID, attemptCount, util.DefaultCacheExpiration)

		nextRunTime := time.Now().Add(retryDelay)
		rt.nextRetries.Store(nextRunTime, payload)
	} else {
		return fmt.Errorf("Payload %s is not retryable. Skipping.\n", payload.ID)
	}

	return nil
}

// getterFn is a function that retrieves the retryTick for the given time.
func (rt *retryTicker) getterFn(ctx context.Context, t time.Time) (Tick, error) {
	rt.payloadAttempts.ClearExpired()

	upkeepPayloads := []ocr2keepers.UpkeepPayload{}

	rt.nextRetries.Range(func(key, value interface{}) bool {
		if runTime, ok := key.(time.Time); ok {
			if t.After(runTime) {
				if payload, ok := value.(ocr2keepers.UpkeepPayload); ok {
					upkeepPayloads = append(upkeepPayloads, payload)
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
func NewRetryTicker(interval time.Duration, observer observer) *retryTicker {
	rt := &retryTicker{
		nextRetries:     sync.Map{},
		payloadAttempts: util.NewCache[int](attemptsCacheExpiration),
	}

	rt.timeTicker = NewTimeTicker(interval, observer, rt.getterFn)

	return rt
}

type retryTick struct {
	upkeeps []ocr2keepers.UpkeepPayload
}

// GetUpkeeps returns the upkeeps contained in the retryTick.
func (t retryTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return t.upkeeps, nil
}
