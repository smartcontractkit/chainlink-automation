package tickers

import (
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type recoveryTicker struct {
	*retryTicker
	// lastBlock ocr2keepers.BlockKey
}

// getterFn is a function that retrieves the recoveryTick for the given time.
func (rt *recoveryTicker) payloadModFunc(p ocr2keepers.UpkeepPayload) ocr2keepers.UpkeepPayload {
	// TODO: update block in payload
	return p
}

// NewRetryTicker creates a new retryTicker with the specified interval and observer.
func NewRecoveryTicker(interval time.Duration, observer observer, configFuncs ...RetryConfigFunc) *recoveryTicker {
	rt := NewRetryTicker(interval, observer, configFuncs...)

	rct := &recoveryTicker{
		retryTicker: rt,
	}

	rt.payloadModFunc = rct.payloadModFunc

	return rct
}
