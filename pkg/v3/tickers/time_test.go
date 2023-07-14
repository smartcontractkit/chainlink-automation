package tickers

import (
	"context"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockObserver struct {
	processFn func(context.Context, Tick[[]int]) error
}

func (o *mockObserver) Process(ctx context.Context, t Tick[[]int]) error {
	return o.processFn(ctx, t)
}

type mockCustomTick struct {
	getFn func(ctx context.Context) ([]int, error)
}

func (t *mockCustomTick) Value(ctx context.Context) ([]int, error) {
	return t.getFn(ctx)
}

func TestNewTimeTicker(t *testing.T) {
	t.Run("creates new time ticker with a counting observer", func(t *testing.T) {
		var mu sync.RWMutex
		callCount := 0

		observr := &mockObserver{
			processFn: func(ctx context.Context, t Tick[[]int]) error {
				mu.Lock()
				defer mu.Unlock()

				callCount++

				return nil
			},
		}

		getFn := func(ctx context.Context, t time.Time) (Tick[[]int], error) {
			return &mockCustomTick{
				getFn: func(ctx context.Context) ([]int, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker[[]int](100*time.Millisecond, observr, getFn, log.New(io.Discard, "", 0))
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		mu.RLock()
		assert.Equal(t, callCount, 4)
		mu.RUnlock()

		time.Sleep(200 * time.Millisecond)

		mu.RLock()
		assert.Equal(t, callCount, 4)
		mu.RUnlock()
	})

	t.Run("creates new time ticker with a processing observer", func(t *testing.T) {
		var mu sync.RWMutex
		callCount := 0

		expected := []int{5, 6}

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick[[]int]) error {
				mu.Lock()
				defer mu.Unlock()

				callCount++

				values, err := tick.Value(ctx)
				if err != nil {
					return err
				}

				if !reflect.DeepEqual(values, expected) {
					t.Fatal("unexpected payloads")
				}
				return nil
			},
		}

		getFn := func(ctx context.Context, t time.Time) (Tick[[]int], error) {
			return &mockCustomTick{
				getFn: func(ctx context.Context) ([]int, error) {
					return expected, nil
				},
			}, nil
		}

		ticker := NewTimeTicker[[]int](100*time.Millisecond, observr, getFn, log.New(io.Discard, "", 0))
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		mu.RLock()
		assert.Equal(t, callCount, 4)
		mu.RUnlock()
	})

	t.Run("creates a ticker with an observer that errors when the getter errors", func(t *testing.T) {
		msg := new(strings.Builder)

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick[[]int]) error {
				return errors.New("boom")
			},
		}

		getFn := func(ctx context.Context, t time.Time) (Tick[[]int], error) {
			return nil, errors.New("error fetching tick")
		}

		ticker := NewTimeTicker[[]int](100*time.Millisecond, observr, getFn, log.New(msg, "", log.LstdFlags))

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
			wg.Done()
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		wg.Wait()

		assert.Contains(t, msg.String(), "error processing observer: boom")
	})

	t.Run("creates a ticker with an observer that errors on processing", func(t *testing.T) {
		msg := new(strings.Builder)

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick[[]int]) error {
				return errors.New("process error")
			},
		}

		getFn := func(ctx context.Context, t time.Time) (Tick[[]int], error) {
			return &mockCustomTick{
				getFn: func(ctx context.Context) ([]int, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker[[]int](100*time.Millisecond, observr, getFn, log.New(msg, "", log.LstdFlags))

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
			wg.Done()
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		wg.Wait()
		assert.Contains(t, msg.String(), "error processing observer: process error")
	})
}
