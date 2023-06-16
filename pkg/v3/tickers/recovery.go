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
	ErrNotRecoverable           = fmt.Errorf("payload is not recoverable")
	ErrRecoveryDurationExceeded = fmt.Errorf("payload has exceeded recovery window")
)

const (
	DefaultRecoveryDelay           = 30 * time.Minute
	DefaultMaxRecoveryDuration     = 24 * time.Hour
	DefaultRecoveryCacheExpiration = 2 * DefaultMaxRecoveryDuration // 10 minutes
)

type RecoveryConfig struct {
	RecoveryDelay           time.Duration
	MaxRecoveryDuration     time.Duration
	RecoveryCacheExpiration time.Duration
}

type RecoveryConfigFunc func(*RecoveryConfig)

var RecoveryWithDefaults = func(c *RecoveryConfig) {
	c.RecoveryDelay = DefaultRecoveryDelay
	c.MaxRecoveryDuration = DefaultMaxRecoveryDuration
	c.RecoveryCacheExpiration = DefaultRecoveryCacheExpiration
}

type recoveryTicker struct {
	timeTicker
	config          *RecoveryConfig
	nextRecoveries  sync.Map               // ocr2keepers.UpkeepPayload -> time.Time
	recoveryEntries *util.Cache[time.Time] // ocr2keepers.UpkeepPayload -> time.Time
}

// Recover adds a retryable result to the recoveryTicker.
func (rt *recoveryTicker) Recover(result ocr2keepers.CheckResult) error {
	payload := result.Payload

	if !result.Recoverable {
		// exit condition for not retryable
		return fmt.Errorf("%w: %s", ErrNotRecoverable, payload.ID)
	}

	entryTime, ok := rt.recoveryEntries.Get(payload.ID)
	if !ok {
		entryTime = time.Now()
		rt.recoveryEntries.Set(payload.ID, entryTime, util.DefaultCacheExpiration)
	}

	now := time.Now()
	if now.Sub(entryTime) > rt.config.MaxRecoveryDuration {
		// exit condition for exceeding maximum recovery time
		return fmt.Errorf("%w: %s", ErrRecoveryDurationExceeded, payload.ID)
	}

	// TODO set block key for this payload
	rt.nextRecoveries.Store(payload, now.Add(rt.config.RecoveryDelay))

	return nil
}

// getterFn is a function that retrieves the recoveryTick for the given time.
func (rt *recoveryTicker) getterFn(ctx context.Context, t time.Time) (Tick, error) {
	rt.recoveryEntries.ClearExpired()

	upkeepPayloads := []ocr2keepers.UpkeepPayload{}

	rt.nextRecoveries.Range(func(key, value interface{}) bool {
		if payload, ok := key.(ocr2keepers.UpkeepPayload); ok {
			if runTime, ok := value.(time.Time); ok {
				if t.After(runTime) {
					upkeepPayloads = append(upkeepPayloads, payload)
					rt.nextRecoveries.Delete(payload)
				}
			}
		}

		return true
	})

	return recoveryTick{
		upkeeps: upkeepPayloads,
	}, nil
}

// NewRetryTicker creates a new retryTicker with the specified interval and observer.
func NewRecoveryTicker(interval time.Duration, observer observer, configFuncs ...RecoveryConfigFunc) *recoveryTicker {
	config := &RecoveryConfig{}

	if len(configFuncs) == 0 {
		RecoveryWithDefaults(config)
	} else {
		for _, f := range configFuncs {
			f(config)
		}
	}

	rt := &recoveryTicker{
		config:          config,
		nextRecoveries:  sync.Map{},
		recoveryEntries: util.NewCache[time.Time](config.RecoveryCacheExpiration),
	}

	rt.timeTicker = *NewTimeTicker(interval, observer, rt.getterFn)

	return rt
}

type recoveryTick struct {
	upkeeps []ocr2keepers.UpkeepPayload
}

// GetUpkeeps returns the upkeeps contained in the retryTick.
func (t recoveryTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return t.upkeeps, nil
}
