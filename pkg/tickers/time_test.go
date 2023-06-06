package time

import (
	"context"
	"fmt"
	"reflect"
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

type mockUpkeepPayload struct {
	data string
}

type mockTick struct {
	getUpkeepsFn func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error)
}

func (t *mockTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return t.getUpkeepsFn(ctx)
}

func TestNewTimeTicker(t *testing.T) {
	t.Run("creates new time ticker with a counting observer", func(t *testing.T) {
		callCount := 0
		observr := &mockObserver{
			processFn: func(ctx context.Context, t Tick) error {
				callCount++
				return nil
			},
		}
		upkeepsFn := func(ctx context.Context, t time.Time) (Tick, error) {
			return &mockTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)

		time.Sleep(450 * time.Millisecond)

		ticker.Stop()

		assert.Equal(t, callCount, 4)
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, callCount, 4)
	})

	t.Run("creates new time ticker with a processing observer", func(t *testing.T) {
		callCount := 0

		upkeepPayloads := []ocr2keepers.UpkeepPayload{
			&mockUpkeepPayload{
				data: "first mock data payload",
			},
			&mockUpkeepPayload{
				data: "second mock data payload",
			},
		}

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
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
			return &mockTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return upkeepPayloads, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)

		time.Sleep(450 * time.Millisecond)

		ticker.Stop()

		assert.Equal(t, callCount, 4)
	})

	t.Run("creates a ticker with an observer that errors on processing", func(t *testing.T) {
		var msg string
		oldLogPrintf := logPrintf
		logPrintf = func(format string, v ...any) {
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
			return &mockTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)

		time.Sleep(450 * time.Millisecond)

		ticker.Stop()

		assert.Equal(t, msg, "error processing observer: boom")
	})

	t.Run("creates a ticker with an observer that exceeds the processing timeout", func(t *testing.T) {
		successfulCallCount := 0

		var msg string
		oldLogPrintf := logPrintf
		logPrintf = func(format string, v ...any) {
			msg = fmt.Sprintf(format, v...)
		}
		defer func() {
			logPrintf = oldLogPrintf
		}()

		firstRun := true

		observr := &mockObserver{
			processFn: func(ctx context.Context, tick Tick) error {
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
			return &mockTick{
				getUpkeepsFn: func(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
					return nil, nil
				},
			}, nil
		}

		ticker := NewTimeTicker(100*time.Millisecond, observr, upkeepsFn)

		time.Sleep(450 * time.Millisecond)

		ticker.Stop()

		assert.Equal(t, msg, "error processing observer: context deadline exceeded")
		assert.Equal(t, successfulCallCount, 3)
	})
}
