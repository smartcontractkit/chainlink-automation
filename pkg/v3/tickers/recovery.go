package tickers

import (
	"sync"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

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
func NewRecoveryTicker(interval time.Duration, observer observer, configFuncs ...RetryConfigFunc) *recoveryTicker {
	rt := NewRetryTicker(interval, observer, configFuncs...)

	rct := &recoveryTicker{
		retryTicker: rt,
	}

	rt.payloadModFunc = rct.modifyPayload

	return rct
}
