package tickers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

func TestRecoveryTicker(t *testing.T) {
	t.Run("sends retryable or recoverable tick to observer after wait", func(t *testing.T) {
		// Create a retryable CheckResult
		retryableResult1 := ocr2keepers.CheckResult{
			Payload:   ocr2keepers.UpkeepPayload{ID: "retryable_1"},
			Retryable: true,
		}

		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			count int
		)

		// create a mocked observer that tracks retries and mocks pipeline results
		mockObserver := &mockObserver{
			processFn: func(ctx context.Context, t Tick) error {
				upkeeps, _ := t.GetUpkeeps(ctx)

				// assert that retry was in tick at least once
				mu.Lock()
				count += len(upkeeps)
				mu.Unlock()

				// TODO: assert that block number is updated for each tick

				return nil
			},
		}

		// set some short time values to confine the tests, recover every 50ms
		config := func(c *RetryConfig) {
			c.RetryDelay = 50 * time.Millisecond
			c.MaxRetryDuration = 250 * time.Millisecond
		}

		// Create a recoveryTicker instance which ticks every 10ms
		rt := NewRecoveryTicker(10*time.Millisecond, mockObserver, RetryWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send 1 recoverable result to the ticker
		assert.NoError(t, rt.Retry(retryableResult1))

		// wait a little longer(60ms) than the recovery delay(50ms)
		time.Sleep(60 * time.Millisecond)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 1, count, "tick should have been retried exactly once")
	})
}
