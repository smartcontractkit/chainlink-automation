package util

import "time"

// CacheCleaner is an interface for cleaning a cache
type CacheCleaner interface {
	// Start starts the cleaning interval
	Start(cache CleanableCache)
	// Stop stops the cleaning interval
	Stop()
}

// CleanableCache is an interface for a cache that can be cleaned
type CleanableCache interface {
	ClearExpired()
}

type cacheCleaner struct {
	interval time.Duration
	stop     chan struct{}
}

func NewCacheCleaner(interval time.Duration) *cacheCleaner {
	return &cacheCleaner{
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (c *cacheCleaner) Start(cache CleanableCache) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cache.ClearExpired()
		case <-c.stop:
			return
		}
	}
}

func (c *cacheCleaner) Stop() {
	close(c.stop)
}
