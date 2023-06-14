package tickers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

func TestRetryTicker_Retry(t *testing.T) {
	mockObserver := &mockObserver{
		processFn: func(ctx context.Context, t Tick) error {
			return nil
		},
	}
	// Create a retryTicker instance
	rt := NewRetryTicker(1*time.Second, mockObserver)
	go func() {
		assert.NoError(t, rt.Start(context.Background()))
	}()

	// Create a retryable CheckResult
	retryableResult1 := ocr2keepers.CheckResult{
		Payload:   ocr2keepers.UpkeepPayload{ID: "retryable_1"},
		Retryable: true,
	}

	// Retry the result
	err := rt.Retry(retryableResult1)
	assert.NoError(t, err)

	// Assert that the retryTicker contains the retryable payload
	//assert.Equal(t, 1, len(rt.nextRetries))
	assert.Equal(t, 1, rt.nextRetriesLen())
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	// Retry second time
	err = rt.Retry(retryableResult1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	// Retry third time
	err = rt.Retry(retryableResult1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	// Retry the result again, it should be skipped due to reaching the maximum attempt count
	err = rt.Retry(retryableResult1)
	// should return an error
	assert.ErrorIs(t, err, ErrTooManyRetries)
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	// Create another retryable CheckResult
	retryableResult2 := ocr2keepers.CheckResult{
		Payload:   ocr2keepers.UpkeepPayload{ID: "retryable_2"},
		Retryable: true,
	}

	// Wait for 8 seconds, retry attempts cache for the above payload should expire(default is 5 seconds)
	time.Sleep(8 * time.Second)

	// Retry another payload
	err = rt.Retry(retryableResult2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	// Create a non-retryable CheckResult
	nonRetryableResult := ocr2keepers.CheckResult{
		Payload:   ocr2keepers.UpkeepPayload{ID: "non-retryable"},
		Retryable: false,
	}

	// Retry the non-retryable result, it should return an error
	err = rt.Retry(nonRetryableResult)
	assert.Error(t, err)

	// Assert that the above non-retryable payload is not added to retryTicker
	assert.Equal(t, 1, len(rt.payloadAttempts.Keys()))

	assert.NoError(t, rt.Close())
}

func TestRetryTicker_getterFn(t *testing.T) {
	// Create a retryTicker instance
	rt := NewRetryTicker(1*time.Second, nil)
	go func() {
		assert.NoError(t, rt.Start(context.Background()))
	}()

	// Add a retryable payload to the retryTicker
	payload := ocr2keepers.UpkeepPayload{ID: "payload1"}
	rt.nextRetries.Store(time.Now().Add(-1*time.Second), payload)

	// Call getterFn to retrieve the retryTick
	tick, err := rt.getterFn(context.Background(), time.Now())
	assert.NoError(t, err)

	// Assert that the retrieved upkeeps match the added payload
	assert.Equal(t, []ocr2keepers.UpkeepPayload{payload}, tick.(retryTick).upkeeps)

	// Assert that the retryTicker is empty after retrieval
	assert.Equal(t, 0, rt.nextRetriesLen())
	assert.NoError(t, rt.Close())
}

func TestNewRetryTicker(t *testing.T) {
	// Create a retryTicker instance
	rt := NewRetryTicker(1*time.Second, nil)
	go func() {
		assert.NoError(t, rt.Start(context.Background()))
	}()

	// Assert that the retryTicker is initialized correctly
	assert.NotNil(t, rt.timeTicker)
	assert.Equal(t, 0, rt.nextRetriesLen())
	assert.NotNil(t, rt.payloadAttempts)
	assert.NoError(t, rt.Close())
}

func TestRetryTick_GetUpkeeps(t *testing.T) {
	// Create a retryTick instance
	upkeeps := []ocr2keepers.UpkeepPayload{
		{ID: "payload1"},
		{ID: "payload2"},
	}
	tick := retryTick{upkeeps: upkeeps}

	// Call GetUpkeeps to retrieve the upkeeps
	retrievedUpkeeps, err := tick.GetUpkeeps(context.Background())

	// Assert that the retrieved upkeeps match the original upkeeps
	assert.NoError(t, err)
	assert.Equal(t, upkeeps, retrievedUpkeeps)
}

// Helper function to get the length of the nextRetries sync.Map
func (rt *retryTicker) nextRetriesLen() int {
	len := 0
	rt.nextRetries.Range(func(_, _ interface{}) bool {
		len++
		return true
	})
	return len
}
