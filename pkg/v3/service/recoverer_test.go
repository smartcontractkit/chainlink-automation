package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceRecoverer(t *testing.T) {
	t.Run("creates and starts a recoverable service", func(t *testing.T) {
		startCh := make(chan struct{}, 1)
		stopCh := make(chan struct{}, 1)
		wrappedService := &testService{
			StartFn: func(_ context.Context) error {
				startCh <- struct{}{}
				return nil
			},
			CloseFn: func() error {
				stopCh <- struct{}{}
				return nil
			},
		}

		svc := NewServiceRecoverer(wrappedService, log.Default())
		svc.coolDown = time.Millisecond

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			svc.Start(context.Background())
			wg.Done()
		}()

		<-startCh

		svc.Close()

		<-stopCh

		wg.Wait()
	})

	t.Run("the underlying service errors and is recovered", func(t *testing.T) {
		ch := make(chan struct{})

		svc := NewServiceRecoverer(&testService{
			StartFn: func(_ context.Context) error {
				ch <- struct{}{}
				return errors.New("boom")
			},
			CloseFn: func() error {
				return nil
			},
		}, log.Default())

		svc.coolDown = 10 * time.Millisecond

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			svc.Start(context.Background())
			wg.Done()
		}()

		<-ch

		svc.Close()

		wg.Wait()
	})

	t.Run("the underlying service panics and is recovered", func(t *testing.T) {
		callCount := atomic.Int32{}
		ch := make(chan struct{})

		rec := NewServiceRecoverer(&testService{
			StartFn: func(_ context.Context) error {
				callCount.Add(1)
				if callCount.Load() == 1 {
					panic("something worth panicking over")
				} else if callCount.Load() > 1 {
					ch <- struct{}{}
				}
				return nil
			},
			CloseFn: func() error {
				return nil
			},
		}, log.Default())

		rec.coolDown = 10 * time.Millisecond

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			rec.Start(context.Background())
			wg.Done()
		}()

		<-ch

		rec.Close()
		assert.Equal(t, callCount.Load(), int32(2))

		wg.Wait()
	})
}

type testService struct {
	StartFn func(context.Context) error
	CloseFn func() error
}

func (t *testService) Start(ctx context.Context) error {
	return t.StartFn(ctx)
}

func (t *testService) Close() error {
	return t.CloseFn()
}
