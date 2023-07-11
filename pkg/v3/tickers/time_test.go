package tickers

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type mockObserver struct {
	processFn func(context.Context, Tick) error
}

func (o *mockObserver) Process(ctx context.Context, t Tick) error {
	return o.processFn(ctx, t)
}

type mockCustomTick struct {
	getUpkeepsFn func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error)
}

func (t *mockCustomTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return t.getUpkeepsFn(ctx)
}

func TestNewTimeTicker(t *testing.T) {
	t.Run("creates new time ticker with a counting observer", func(t *testing.T) {
		var mu sync.RWMutex
		callCount := 0

		observr := &mockObserver{
			processFn: func(ctx context.Context, t Tick) error {
				mu.Lock()
				defer mu.Unlock()

				callCount++

				return nil
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return &mockCustomTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)
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

		upkeepPayloads := []ocr2keepers.UpkeepPayload{
			{
				ID: "first mock data payload",
			},
			{
				ID: "second mock data payload",
			},
		}

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
				mu.Lock()
				defer mu.Unlock()

				callCount++

				upkeeps, err := tick.GetUpkeeps(ctx)
				if err != nil {
					return err
				}

				if !reflect.DeepEqual(upkeeps, upkeepPayloads) {
					t.Fatal("unexpected payloads")
				}
				return nil
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return &mockCustomTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return upkeepPayloads, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)
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
		var mu sync.RWMutex
		var msg string

		oldLogPrintf := logPrintf
		logPrintf = func(format string, v ...any) {
			mu.Lock()
			defer mu.Unlock()

			msg = fmt.Sprintf(format, v...)
		}
		defer func() {
			logPrintf = oldLogPrintf
		}()

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
				return errors.New("boom")
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return nil, errors.New("error fetching tick")
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		mu.RLock()
		assert.Equal(t, msg, "error processing observer: boom")
		mu.RUnlock()
	})

	t.Run("creates a ticker with an observer that errors on processing", func(t *testing.T) {
		var mu sync.RWMutex
		var msg string

		oldLogPrintf := logPrintf
		logPrintf = func(format string, v ...any) {
			mu.Lock()
			defer mu.Unlock()

			msg = fmt.Sprintf(format, v...)
		}
		defer func() {
			logPrintf = oldLogPrintf
		}()

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
				return errors.New("process error")
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return &mockCustomTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		mu.RLock()
		assert.Equal(t, msg, "error processing observer: process error")
		mu.RUnlock()
	})

	t.Run("creates a ticker with an observer that exceeds the processing timeout", func(t *testing.T) {
		successfulCallCount := 0

		var mu sync.RWMutex
		var msg string

		oldLogPrintf := logPrintf
		logPrintf = func(format string, v ...any) {
			mu.Lock()
			defer mu.Unlock()

			msg = fmt.Sprintf(format, v...)
		}
		defer func() {
			logPrintf = oldLogPrintf
		}()

		firstRun := true

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
				mu.Lock()
				defer mu.Unlock()

				if firstRun {
					firstRun = false
					<-ctx.Done()
					return ctx.Err()
				}
				successfulCallCount++
				return nil
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return &mockCustomTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)
		go func() {
			assert.NoError(t, ticker.Start(context.Background()))
		}()

		time.Sleep(450 * time.Millisecond)

		assert.NoError(t, ticker.Close())

		mu.RLock()
		assert.Equal(t, msg, "error processing observer: context deadline exceeded")
		assert.Equal(t, successfulCallCount, 3)
		mu.RUnlock()
	})
}
