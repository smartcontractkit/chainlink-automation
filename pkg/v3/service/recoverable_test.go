package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testService struct {
	DoFn   func(context.Context) error
	StopFn func() error
}

func (t *testService) Start(ctx context.Context) error {
	return t.DoFn(ctx)
}

func (t *testService) Close() error {
	return t.StopFn()
}

func TestNewRecoverableService(t *testing.T) {
	t.Run("creates and starts a recoverable service", func(t *testing.T) {
		svc := NewRecoverer(&testService{
			DoFn: func(_ context.Context) error {
				return nil
			},
			StopFn: func() error {
				return nil
			},
		}, log.Default())

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			assert.NoError(t, svc.Start(context.Background()), "the service should start without error")
			wg.Done()
		}()

		time.Sleep(50 * time.Millisecond)

		assert.True(t, svc.running.Load(), "the service should be running")
		assert.NoError(t, svc.Close(), "no error on close")

		time.Sleep(50 * time.Millisecond)

		assert.False(t, svc.running.Load(), "the service should be stopped")

		wg.Wait()
	})

	t.Run("should not be able to start an already running service", func(t *testing.T) {
		svc := NewRecoverer(&testService{
			DoFn: func(_ context.Context) error {
				t.Fatal("do should not be called when the service is already running")
				return nil
			},
			StopFn: func() error {
				return nil
			},
		}, log.Default())

		svc.running.Store(true)

		assert.ErrorIs(t, svc.Start(context.Background()), ErrServiceAlreadyStarted)
		assert.True(t, svc.running.Load(), "running state should still be true")

		assert.NoError(t, svc.Close(), "no error in calling close")
	})

	t.Run("should not be able to stop an already stopped service", func(t *testing.T) {
		svc := NewRecoverer(&testService{
			DoFn: func(_ context.Context) error {
				return nil
			},
			StopFn: func() error {
				t.Fatal("do should not be called when the service is already running")
				return nil
			},
		}, log.Default())

		// should be default but set it to false anyway
		svc.running.Store(false)

		assert.ErrorIs(t, svc.Close(), ErrServiceNotRunning)
		assert.False(t, svc.running.Load())
	})

	t.Run("a running service is stopped by the underlying service returning an error", func(t *testing.T) {
		t.Skip() // TODO skip this flake for now

		callCount := atomic.Int32{}
		ch := make(chan struct{})

		svc := NewRecoverer(&testService{
			DoFn: func(_ context.Context) error {
				callCount.Add(1)
				if callCount.Load() == 1 {
					return errServiceStopped
				} else if callCount.Load() > 1 {
					ch <- struct{}{}
				}
				return nil
			},
			StopFn: func() error {
				return nil
			},
		}, log.Default())

		svc.coolDown = 10 * time.Millisecond

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			assert.NoError(t, svc.Start(context.Background()))
			wg.Done()
		}()

		<-ch

		assert.NoError(t, svc.Close())
		assert.Equal(t, callCount.Load(), int32(2))

		wg.Wait()
	})

	t.Run("a running service is stopped by the underlying service causing a panic", func(t *testing.T) {
		t.Skip() // TODO skip this flake for now
		callCount := 0
		ch := make(chan struct{})

		svc := NewRecoverer(&testService{
			DoFn: func(_ context.Context) error {
				callCount++
				if callCount == 1 {
					panic("something worth panicking over")
				} else if callCount > 1 {
					ch <- struct{}{}
				}
				return nil
			},
			StopFn: func() error {
				return nil
			},
		}, log.Default())

		svc.coolDown = 10 * time.Millisecond

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			assert.NoError(t, svc.Start(context.Background()))
			wg.Done()
		}()

		<-ch

		assert.NoError(t, svc.Close())
		assert.Equal(t, callCount, 2)

		wg.Wait()
	})
}
