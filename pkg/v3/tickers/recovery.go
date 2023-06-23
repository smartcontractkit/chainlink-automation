package tickers

import (
	"log"
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

const (
	DefaultRecoveryDelay           = 1 * time.Minute
	DefaultMaxRecoveryDuration     = 60 * time.Minute
	DefaultRecoveryCacheExpiration = 2 * DefaultMaxRecoveryDuration
)

var RecoveryWithDefaults = func(c *RetryConfig) {
	c.RetryDelay = DefaultRecoveryDelay
	c.MaxRetryDuration = DefaultMaxRecoveryDuration
	c.RetryCacheExpiration = DefaultRecoveryCacheExpiration
}

type recoveryTicker struct {
	*retryTicker
	lock      sync.RWMutex
	lastBlock ocr2keepers.BlockKey
}

// setBlock updates the block of the given payload
func (rt *recoveryTicker) SetBlock(key ocr2keepers.BlockKey) {
	rt.lock.Lock()
	defer rt.lock.Unlock()
	rt.lastBlock = key
}

// modifyPayload updates and returns the payload
func (rt *recoveryTicker) modifyPayload(p ocr2keepers.UpkeepPayload) ocr2keepers.UpkeepPayload {
	rt.lock.RLock()
	defer rt.lock.RUnlock()
	p.CheckBlock = rt.lastBlock
	return p
}

// NewRetryTicker creates a new retryTicker with the specified interval and observer.
func NewRecoveryTicker(interval time.Duration, observer observer, logger *log.Logger, configFuncs ...RetryConfigFunc) *recoveryTicker {
	rt := NewRetryTicker(interval, observer, logger, configFuncs...)

	rct := &recoveryTicker{
		retryTicker: rt,
	}

	rt.payloadModFunc = rct.modifyPayload

	return rct
}
