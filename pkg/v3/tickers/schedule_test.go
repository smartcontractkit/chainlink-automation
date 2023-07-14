package tickers

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

func TestScheduleTicker(t *testing.T) {
	t.Run("sends scheduled tick to observer after wait", func(t *testing.T) {
		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			count int
		)

		// create a mocked observer that tracks sends and mocks results
		mockObserver := &mockObserver{
			processFn: func(ctx context.Context, t Tick[[]int]) error {
				values, _ := t.Value(ctx)

				// assert that value was in tick at least once
				mu.Lock()
				count += len(values)
				mu.Unlock()

				return nil
			},
		}

		importFunc := func(func(string, int) error) error {
			return nil
		}

		// set some short time values to confine the tests
		config := func(c *ScheduleTickerConfig) {
			c.SendDelay = 50 * time.Millisecond
			c.MaxSendDuration = 250 * time.Millisecond
		}

		// Create a scheduled ticker instance
		rt := NewScheduleTicker[int](10*time.Millisecond, mockObserver, importFunc, log.New(io.Discard, "", 0), ScheduleTickerWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send the value
		assert.NoError(t, rt.Schedule("test", 6))

		// wait a little longer than the send delay
		time.Sleep(60 * time.Millisecond)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 1, count, "tick should have been sent exactly once")
	})

	t.Run("does not send value before wait", func(t *testing.T) {
		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			count int
		)

		// create a mocked observer that tracks sends and mocks results
		mockObserver := &mockObserver{
			processFn: func(ctx context.Context, t Tick[[]int]) error {
				values, _ := t.Value(ctx)

				// assert that value was in tick at least once
				mu.Lock()
				count += len(values)
				mu.Unlock()

				return nil
			},
		}

		importFunc := func(func(string, int) error) error {
			return nil
		}

		// set some short time values to confine the tests
		config := func(c *ScheduleTickerConfig) {
			c.SendDelay = 100 * time.Millisecond
			c.MaxSendDuration = 250 * time.Millisecond
		}

		// Create a schuduled ticker instance
		rt := NewScheduleTicker[int](25*time.Millisecond, mockObserver, importFunc, log.New(io.Discard, "", 0), ScheduleTickerWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send the value
		assert.NoError(t, rt.Schedule("test", 5))

		// wait a little shorter than the send delay
		time.Sleep(50 * time.Millisecond)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 0, count, "tick should not have been sent")
	})

	t.Run("does not allow send after max duration", func(t *testing.T) {
		var (
			wg    sync.WaitGroup
			mu    sync.Mutex
			count int
		)

		// create a mocked observer that tracks sends and mocks results
		mockObserver := &mockObserver{
			processFn: func(ctx context.Context, t Tick[[]int]) error {
				upkeeps, _ := t.Value(ctx)

				// assert that send was in tick at least once
				mu.Lock()
				count += len(upkeeps)
				mu.Unlock()

				return nil
			},
		}

		importFunc := func(func(string, int) error) error {
			return nil
		}

		// set some short time values to confine the tests
		config := func(c *ScheduleTickerConfig) {
			c.SendDelay = 50 * time.Millisecond
			c.MaxSendDuration = 250 * time.Millisecond
		}

		// Create a scheduled ticker instance
		rt := NewScheduleTicker[int](10*time.Millisecond, mockObserver, importFunc, log.New(io.Discard, "", 0), ScheduleTickerWithDefaults, config)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, rt.Start(context.Background()))
			wg.Done()
		}()

		// send the first value
		assert.NoError(t, rt.Schedule("test", 5))

		// wait for the send to succeed
		time.Sleep(100 * time.Millisecond)

		// send the value again to ensure the ability to send the same value
		assert.NoError(t, rt.Schedule("test", 5))

		// wait long enough to be more than the max duration
		time.Sleep(200 * time.Millisecond)

		// attempting another send should return an error
		assert.ErrorIs(t, rt.Schedule("test", 5), ErrSendDurationExceeded)

		assert.NoError(t, rt.Close())

		wg.Wait()

		assert.Equal(t, 2, count, "tick should have been sent exactly twice")
	})
}

func TestStaticTick_Value(t *testing.T) {
	// Create a retryTick instance
	upkeeps := []ocr2keepers.UpkeepPayload{
		{ID: "payload1"},
		{ID: "payload2"},
	}
	tick := staticTick[[]ocr2keepers.UpkeepPayload]{value: upkeeps}

	// Call GetUpkeeps to retrieve the upkeeps
	retrievedUpkeeps, err := tick.Value(context.Background())

	// Assert that the retrieved upkeeps match the original upkeeps
	assert.NoError(t, err)
	assert.Equal(t, upkeeps, retrievedUpkeeps)
}
