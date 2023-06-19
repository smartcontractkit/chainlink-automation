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
	t.Run("sends retryable/recoverable tick to observer after wait", func(t *testing.T) {
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

				return nil
			},
		}

		// set some short time values to confine the tests, recover every 50ms
		config := func(c *RecoveryConfig) {
			c.RecoveryDelay = 50 * time.Millisecond
			c.MaxRecoveryDuration = 250 * time.Millisecond
		}

		// Create a recoveryTicker instance which ticks every 10ms
		rt := NewRecoveryTicker(10*time.Millisecond, mockObserver, RecoveryWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send 1 recoverable result to the ticker
		assert.NoError(t, rt.Recover(retryableResult1))

		// wait a little longer(60ms) than the recovery delay(50ms)
		time.Sleep(60 * time.Millisecond)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 1, count, "tick should have been retried exactly once")
	})

	t.Run("does not send retry before wait", func(t *testing.T) {
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

				return nil
			},
		}

		// set some short time values to confine the tests
		config := func(c *RecoveryConfig) {
			c.RecoveryDelay = 100 * time.Millisecond
			c.MaxRecoveryDuration = 250 * time.Millisecond
		}

		// Create a recoveryTicker instance
		rt := NewRecoveryTicker(25*time.Millisecond, mockObserver, RecoveryWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send the recovery
		assert.NoError(t, rt.Recover(retryableResult1))

		// wait a little shorter(50ms) than the recovery delay(100ms)
		time.Sleep(50 * time.Millisecond)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 0, count, "tick should not have been retried")
	})

	t.Run("does not allow recovery after max duration", func(t *testing.T) {
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

				return nil
			},
		}

		// set some short time values to confine the tests
		config := func(c *RecoveryConfig) {
			c.RecoveryDelay = 50 * time.Millisecond
			c.MaxRecoveryDuration = 250 * time.Millisecond
		}

		// Create a retryTicker instance
		rt := NewRecoveryTicker(10*time.Millisecond, mockObserver, RecoveryWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send the retry
		assert.NoError(t, rt.Recover(retryableResult1))

		// wait for the recover to succeed
		time.Sleep(100 * time.Millisecond)

		// send the retry again to ensure the ability to retry the same value
		assert.NoError(t, rt.Recover(retryableResult1))

		// wait long enough to be more than the max duration
		time.Sleep(200 * time.Millisecond)

		// attempting a retry should return an error
		assert.ErrorIs(t, rt.Recover(retryableResult1), ErrRecoveryDurationExceeded)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 2, count, "tick should have been retried exactly twice")
	})
}

func TestRecoveryTick_GetUpkeeps(t *testing.T) {
	// Create a retryTick instance
	upkeeps := []ocr2keepers.UpkeepPayload{
		{ID: "payload1"},
		{ID: "payload2"},
	}
	tick := recoveryTick{upkeeps: upkeeps}

	// Call GetUpkeeps to retrieve the upkeeps
	retrievedUpkeeps, err := tick.GetUpkeeps(context.Background())

	// Assert that the retrieved upkeeps match the original upkeeps
	assert.NoError(t, err)
	assert.Equal(t, upkeeps, retrievedUpkeeps)
}
