package keepers

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	c := newCache[int](time.Second)

	assert.Equal(t, time.Second, c.defaultExpiration, "must set default expiration from provided value")
	assert.Equal(t, make(map[string]cacheItem[int]), c.data, "must initialize empty data value")
}

func TestCacheSet(t *testing.T) {
	tests := []struct {
		Name       string
		Key        string
		Value      int
		Expiration time.Duration
	}{
		{Name: "Default Expire", Key: "key1", Value: 10, Expiration: defaultExpiration},
		{Name: "Custom Expire", Key: "key2", Value: 50, Expiration: 3 * time.Minute},
		{Name: "Overwrite Key", Key: "key1", Value: 40, Expiration: 3 * time.Minute},
	}

	c := newCache[int](20 * time.Millisecond)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			n := time.Now()
			c.Set(test.Key, test.Value, test.Expiration)

			value, ok := c.data[test.Key]
			assert.Equal(t, true, ok, "key should exist")
			assert.Equal(t, test.Value, value.Item, "cached value should match expected")
			assert.GreaterOrEqual(t, value.Expires, n.Add(test.Expiration).UnixNano(), "expiration should be set")
		})
	}

	assert.Equal(t, 2, len(c.data), "cache should contain 2 keys")
}

func TestCacheGet(t *testing.T) {
	tests := []struct {
		Name       string
		Key        string
		Value      int
		Expiration time.Duration
		Expected   bool
	}{
		{Name: "Not Expired", Key: "key1", Value: 50, Expiration: 3 * time.Minute, Expected: true},
		{Name: "Expired", Key: "key2", Value: 50, Expiration: 1 * time.Millisecond, Expected: false},
		{Name: "Missing Key", Key: "key3", Expiration: 1 * time.Millisecond, Expected: false},
	}

	c := newCache[int](20 * time.Millisecond)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			n := time.Now()
			c.data[test.Key] = cacheItem[int]{Item: test.Value, Expires: n.Add(test.Expiration).UnixNano()}

			// wait for item to expire
			<-time.After(2 * time.Millisecond)

			// do the test
			value, ok := c.Get(test.Key)

			assert.Equal(t, test.Expected, ok, "returned key status should match expected")
			if test.Expected {
				assert.Equal(t, test.Value, value, "cached value should match expected")
			}
		})
	}
}

func TestCacheClearExpired(t *testing.T) {
	c := newCache[int](1 * time.Millisecond)
	n := time.Now()

	// add values that expire quickly
	for i := 1; i <= 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.data[key] = cacheItem[int]{Item: 10 * i, Expires: n.Add(1 * time.Millisecond).UnixNano()}
	}

	// add values that expire slowly
	for i := 6; i <= 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		c.data[key] = cacheItem[int]{Item: 10 * i, Expires: n.Add(1 * time.Minute).UnixNano()}
	}

	// wait for items to expire
	<-time.After(2 * time.Millisecond)

	c.ClearExpired()

	assert.Equal(t, 5, len(c.data), "expired keys should be removed from the data set")
}

func BenchmarkCacheParallelism(b *testing.B) {
	c := newCache[int](10 * time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := rand.Intn(100)
			key := fmt.Sprintf("key-%d", n)
			if n < 30 {
				// 30% writes
				c.Set(key, 10*n, defaultExpiration)
			} else if n < 90 {
				// 60% reads
				c.Get(key)
			} else {
				// 10% clear expired keys
				c.ClearExpired()
			}
		}
	})
}
