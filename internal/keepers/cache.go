package keepers

import (
	"sync"
	"time"
)

const (
	// convenience value for setting expiration to the default value
	defaultExpiration time.Duration = 0
)

type cacheItem[T any] struct {
	Item    T
	Expires int64
}

type cache[T any] struct {
	defaultExpiration time.Duration
	mu                sync.RWMutex
	data              map[string]cacheItem[T]
}

func newCache[T any](expiration time.Duration) *cache[T] {
	return &cache[T]{
		defaultExpiration: expiration,
		data:              make(map[string]cacheItem[T]),
	}
}

func (c *cache[T]) Set(key string, value T, expire time.Duration) {
	var exp int64
	if expire == defaultExpiration {
		expire = c.defaultExpiration
	}

	if expire > 0 {
		exp = time.Now().Add(expire).UnixNano()
	}

	c.mu.Lock()
	c.data[key] = cacheItem[T]{
		Item:    value,
		Expires: exp,
	}
	c.mu.Unlock()
}

func (c *cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	value, found := c.data[key]
	if !found {
		c.mu.RUnlock()
		return getZero[T](), false
	}

	if value.Expires > 0 {
		if time.Now().UnixNano() > value.Expires {
			c.mu.RUnlock()
			return getZero[T](), false
		}
	}

	c.mu.RUnlock()
	return value.Item, true
}

func (c *cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// ClearExpired loops through all keys and evaluates the value
// expire time. If an item is expired, it is removed from the
// cache. This function places a read lock on the data set and
// only obtains a write lock if needed.
func (c *cache[T]) ClearExpired() {
	now := time.Now().UnixNano()
	c.mu.RLock()
	toclear := make([]string, 0, len(c.data))
	for k, item := range c.data {
		if item.Expires > 0 && now > item.Expires {
			toclear = append(toclear, k)
		}
	}
	c.mu.RUnlock()

	if len(toclear) > 0 {
		c.mu.Lock()
		for _, k := range toclear {
			delete(c.data, k)
		}
		c.mu.Unlock()
	}
}

func getZero[T any]() T {
	var result T
	return result
}

type intervalCacheCleaner[T any] struct {
	Interval time.Duration
	stop     chan struct{}
}

func (ic *intervalCacheCleaner[T]) Run(c *cache[T]) {
	ticker := time.NewTicker(ic.Interval)
	for {
		select {
		case <-ticker.C:
			c.ClearExpired()
		case <-ic.stop:
			ticker.Stop()
			return
		}
	}
}
