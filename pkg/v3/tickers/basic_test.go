package tickers

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicTicker(t *testing.T) {
	t.Run("sends values for every tick", func(t *testing.T) {
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

		bt := NewBasicTicker[int](
			20*time.Millisecond,
			mockObserver,
			log.New(io.Discard, "", 0),
		)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, bt.Start(context.Background()))
			wg.Done()
		}()

		// send the value
		assert.NoError(t, bt.Add("test5", 5))
		assert.NoError(t, bt.Add("test6", 6))
		time.Sleep(35 * time.Millisecond)
		assert.NoError(t, bt.Add("test7", 7))

		// wait a little longer for all values to be sent
		time.Sleep(30 * time.Millisecond)

		assert.NoError(t, bt.Close())

		wg.Wait()

		assert.Equal(t, 3, count, "tick should have been sent exactly 3 times")
	})

	t.Run("does not resend value", func(t *testing.T) {
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			sent []int
		)

		// create a mocked observer that tracks sends and mocks results
		mockObserver := &mockObserver{
			processFn: func(ctx context.Context, t Tick[[]int]) error {
				values, _ := t.Value(ctx)

				// assert that value was in tick at least once
				mu.Lock()
				sent = append(sent, values...)
				mu.Unlock()

				return nil
			},
		}

		bt := NewBasicTicker[int](
			20*time.Millisecond,
			mockObserver,
			log.New(io.Discard, "", 0),
		)

		// start the ticker in a separate thread
		wg.Add(1)
		go func() {
			assert.NoError(t, bt.Start(context.Background()))
			wg.Done()
		}()

		// send the value
		assert.NoError(t, bt.Add("test5", 5))
		assert.NoError(t, bt.Add("test6", 6))
		time.Sleep(35 * time.Millisecond)
		assert.NoError(t, bt.Add("test7", 7))

		// wait a little longer for all values to be sent
		time.Sleep(30 * time.Millisecond)

		assert.NoError(t, bt.Close())

		wg.Wait()

		assert.Equal(t, 3, len(sent), "tick should have been sent exactly 3 times")
		assert.Equal(t, []int{5, 6, 7}, sent, "sent values should be equal to expected")
	})
}
