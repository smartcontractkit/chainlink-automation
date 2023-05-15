package util

import (
	"io"
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverableService(t *testing.T) {

	d := &testDoable{
		s: make(chan struct{}),
	}
	l := log.New(io.Discard, "", 0)

	svc := NewRecoverableService(d, l)

	svc.Start()
	svc.Stop()

	assert.Equal(t, d.doCount, 1, "do count expected to be 1")
	assert.Equal(t, d.stopCount, 1, "stop count expected to be 1")
}

type testDoable struct {
	mu        sync.Mutex
	doCount   int
	stopCount int
	s         chan struct{}
}

func (d *testDoable) Do() error {
	d.mu.Lock()
	d.doCount++
	d.mu.Unlock()

	<-d.s

	return nil
}

func (d *testDoable) Stop() {
	d.mu.Lock()
	d.stopCount++
	d.mu.Unlock()

	d.s <- struct{}{}
}
