package util

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testService struct {
	DoFn   func() error
	StopFn func()
}

func (t *testService) Do() error {
	return t.DoFn()
}

func (t *testService) Stop() {
	t.StopFn()
}

func TestNewRecoverableService(t *testing.T) {
	t.Run("creates and starts a recoverable service", func(t *testing.T) {
		svc := NewRecoverableService(&testService{
			DoFn: func() error {
				return nil
			},
			StopFn: func() {

			},
		}, log.Default())

		svc.Start()
		assert.True(t, svc.running)
		svc.Stop()
		assert.False(t, svc.running)
	})

	t.Run("should not be able to start an already running service", func(t *testing.T) {
		svc := NewRecoverableService(&testService{
			DoFn: func() error {
				t.Fatal("do should not be called when the service is already running")
				return nil
			},
			StopFn: func() {

			},
		}, log.Default())

		svc.running = true

		svc.Start()
		assert.True(t, svc.running)
		svc.Stop()
		assert.False(t, svc.running)
	})

	t.Run("should not be able to stop an already stopped service", func(t *testing.T) {
		svc := NewRecoverableService(&testService{
			DoFn: func() error {
				return nil
			},
			StopFn: func() {
				t.Fatal("do should not be called when the service is already running")
			},
		}, log.Default())

		svc.running = false

		svc.Stop()
		assert.False(t, svc.running)
	})

	t.Run("a running service is stopped by the underlying service returning an error", func(t *testing.T) {
		callCount := 0
		oldCoolDown := coolDown
		coolDown = time.Millisecond * 10
		defer func() {
			coolDown = oldCoolDown
		}()
		ch := make(chan struct{})
		svc := NewRecoverableService(&testService{
			DoFn: func() error {
				callCount++
				if callCount == 1 {
					return errServiceStopped
				} else if callCount > 1 {
					ch <- struct{}{}
				}
				return nil
			},
			StopFn: func() {
			},
		}, log.Default())

		svc.Start()

		<-ch

		svc.Stop()

		assert.Equal(t, callCount, 2)
	})

	t.Run("a running service is stopped by the underlying service causing a panic", func(t *testing.T) {
		callCount := 0
		oldCoolDown := coolDown
		coolDown = time.Millisecond * 10
		defer func() {
			coolDown = oldCoolDown
		}()
		ch := make(chan struct{})
		svc := NewRecoverableService(&testService{
			DoFn: func() error {
				callCount++
				if callCount == 1 {
					panic("something worth panicking over")
				} else if callCount > 1 {
					ch <- struct{}{}
				}
				return nil
			},
			StopFn: func() {
			},
		}, log.Default())

		svc.Start()

		<-ch

		svc.Stop()

		assert.Equal(t, callCount, 2)
	})
}
