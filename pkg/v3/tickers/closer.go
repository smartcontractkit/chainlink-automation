package tickers

import (
	"context"
	"sync"
)

type closer struct {
	cancel context.CancelFunc
	lock   sync.Mutex
}

func (c *closer) Store(cancel context.CancelFunc) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cancel != nil {
		return false
	}
	c.cancel = cancel
	return true
}

func (c *closer) Close() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
		return true
	}
	return false
}
