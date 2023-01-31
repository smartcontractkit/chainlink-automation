package util

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMergeContexts(t *testing.T) {

	t.Run("Primary Context Cancel", func(t *testing.T) {
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2 := context.Background()

		ctx := MergeContexts(ctx1, ctx2)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			tmr := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				tmr.Stop()
				assert.Error(t, ctx.Err())
			case <-tmr.C:
				assert.Fail(t, "context did not close")
			}
			wg.Done()
		}()

		cancel1()

		wg.Wait()
	})

	t.Run("Secondary Context Cancel", func(t *testing.T) {
		ctx1 := context.Background()
		ctx2, cancel2 := context.WithCancel(context.Background())

		ctx := MergeContexts(ctx1, ctx2)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			tmr := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				tmr.Stop()
				assert.Error(t, ctx.Err())
			case <-tmr.C:
				assert.Fail(t, "context did not close")
			}
			wg.Done()
		}()

		cancel2()

		wg.Wait()
	})

	t.Run("Merged Context Cancel", func(t *testing.T) {
		ctx1 := context.Background()
		ctx2 := context.Background()

		ctx, cancel := MergeContextsWithCancel(ctx1, ctx2)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			tmr := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				tmr.Stop()
				assert.Error(t, ctx.Err())
			case <-tmr.C:
				assert.Fail(t, "context did not close")
			}
			wg.Done()
		}()

		cancel()
		wg.Wait()
	})

	t.Run("Cancel After Primary", func(t *testing.T) {
		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2 := context.Background()

		ctx, cancel := MergeContextsWithCancel(ctx1, ctx2)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			tmr := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				tmr.Stop()
				assert.Error(t, ctx.Err())
			case <-tmr.C:
				assert.Fail(t, "context did not close")
			}
			wg.Done()
		}()

		cancel1()
		<-time.After(50 * time.Millisecond)
		cancel()

		wg.Wait()
	})
}
