package util

import (
	"log"
	"testing"
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
	})
}
