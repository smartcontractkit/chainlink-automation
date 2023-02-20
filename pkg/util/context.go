package util

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type mergedContext struct {
	mu      sync.RWMutex
	mainCtx context.Context
	ctx     context.Context
	done    chan struct{}
	doneExt chan struct{}
	once    sync.Once
	err     error
}

func MergeContexts(mainCtx, ctx context.Context) context.Context {
	c := &mergedContext{mainCtx: mainCtx, ctx: ctx, done: make(chan struct{}), doneExt: make(chan struct{})}
	go c.run()
	return c
}

func MergeContextsWithCancel(mainCtx, ctx context.Context) (context.Context, context.CancelFunc) {
	c := &mergedContext{mainCtx: mainCtx, ctx: ctx, done: make(chan struct{}), doneExt: make(chan struct{})}
	go c.run()
	return c, c.cancel
}

func (c *mergedContext) Done() <-chan struct{} {
	return c.doneExt
}

func (c *mergedContext) Err() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.err
}

func (c *mergedContext) Deadline() (deadline time.Time, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var d time.Time
	d1, ok1 := c.ctx.Deadline()
	d2, ok2 := c.mainCtx.Deadline()
	if ok1 && d1.UnixNano() < d2.UnixNano() {
		d = d1
	} else if ok2 {
		d = d2
	}
	return d, ok1 || ok2
}

func (c *mergedContext) Value(key interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ctx.Value(key)
}

func (c *mergedContext) cancel() {
	c.mu.Lock()
	if c.err == nil {
		c.err = fmt.Errorf("merged context canceled")
	}
	c.mu.Unlock()
	c.once.Do(c.closeChannels)
}

func (c *mergedContext) run() {
	var doneCtx context.Context

	select {
	case <-c.mainCtx.Done():
		doneCtx = c.mainCtx
	case <-c.ctx.Done():
		doneCtx = c.ctx
	case <-c.done:
		break
	}

	c.mu.Lock()
	if c.err == nil {
		c.err = doneCtx.Err()
	}
	c.mu.Unlock()
	c.once.Do(c.closeChannels)
}

func (c *mergedContext) closeChannels() {
	close(c.done)
	close(c.doneExt)
}
