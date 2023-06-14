package resultstore

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/stretchr/testify/assert"
)

func TestResultStore_Sanity(t *testing.T) {
	lggr := log.New(io.Discard, "", 0)

	tests := []struct {
		name          string
		itemsToAdd    []ocr2keepers.CheckResult
		itemsToRemove []string
		expected      []ocr2keepers.CheckResult
	}{
		{
			name:          "happy path",
			itemsToAdd:    mockItems(0, 5),
			itemsToRemove: append(mockIDs(1, 1), mockIDs(3, 1)...),
			expected:      append(mockItems(0, 1), append(mockItems(2, 1), mockItems(4, 1)...)...),
		},
		{
			name:          "remove non-existent item",
			itemsToAdd:    mockItems(0, 2),
			itemsToRemove: []string{"boo"},
			expected:      mockItems(0, 2),
		},
		{
			name:          "no items",
			itemsToAdd:    []ocr2keepers.CheckResult{},
			itemsToRemove: []string{"boo"},
			expected:      []ocr2keepers.CheckResult{},
		},
		{
			name:          "no items to remove",
			itemsToAdd:    mockItems(0, 1),
			itemsToRemove: []string{},
			expected:      mockItems(0, 1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := New(lggr)
			view, err := store.View()
			assert.NoError(t, err)
			assert.Len(t, view, 0)
			store.Add(tc.itemsToAdd...)
			view, err = store.View()
			assert.NoError(t, err)
			assert.Len(t, view, len(tc.itemsToAdd))
			store.Remove(tc.itemsToRemove...)
			view, err = store.View()
			assert.NoError(t, err)
			assert.Len(t, view, len(tc.expected))
			for _, v := range view {
				assert.Contains(t, tc.expected, v)
			}
		})
	}
}

func TestResultStore_GC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lggr := log.New(io.Discard, "", 0)
	store := New(lggr)

	store.Add(mockItems(0, 3)...)

	var notifications []ocr2keepersv3.Notification
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		lctx, cancel := context.WithCancel(ctx)
		defer cancel()

		notify := store.Notifications()
		for {
			select {
			case n := <-notify:
				if n.Op == ocr2keepersv3.NotifyOpNil {
					return
				}
				notifications = append(notifications, n)
			case <-lctx.Done():
				return
			}
		}
	}()

	store.Remove(mockIDs(0, 2)...)

	store.lock.Lock()
	el := store.data["test-id-2"]
	el.addedAt = time.Now().Add(-2 * storeTTL)
	store.data["test-id-2"] = el
	store.lock.Unlock()

	store.gc()

	// using nil notification to signal end of notifications
	store.notifications <- ocr2keepersv3.Notification{
		Op: ocr2keepersv3.NotifyOpNil,
	}

	wg.Wait()

	assert.Len(t, notifications, 3)
	ops := []ocr2keepersv3.NotifyOp{ocr2keepersv3.NotifyOpRemove, ocr2keepersv3.NotifyOpRemove, ocr2keepersv3.NotifyOpEvict}
	for i, notification := range notifications {
		assert.Equal(t, ops[i], notification.Op)
	}
}

func TestResultStore_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lggr := log.New(io.Discard, "", 0)

	store := New(lggr)
	origGcInterval := gcInterval
	origStoreTTL := storeTTL
	defer func() {
		gcInterval = origGcInterval
		storeTTL = origStoreTTL
	}()
	storeTTL = time.Millisecond * 2
	gcInterval = time.Millisecond * 5

	go func() {
		defer func() {
			_ = store.Close()
		}()
		if err := store.Start(ctx); err != nil {
			panic(err)
		}
	}()
	store.Add(mockItems(0, 2)...)
	view, err := store.View()
	assert.NoError(t, err)
	assert.Len(t, view, 2)

	<-time.After(gcInterval * 2)

	view, err = store.View()
	assert.NoError(t, err)
	assert.Len(t, view, 0)
}

func TestResultStore_Concurrency(t *testing.T) {
	lggr := log.New(io.Discard, "", 0)
	store := New(lggr)

	workers := 4
	nitems := int32(1000)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		doneWrite := make(chan bool)
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			items := int(atomic.LoadInt32(&nitems))
			n := items * (i + 1)
			for j := items * i; j < n; j++ {
				store.Add(mockItems(j, 1)...)
			}
			doneWrite <- true
		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-doneWrite
			items := int(atomic.LoadInt32(&nitems))
			n := items * (i + 1)
			for j := items * i; j < n; j++ {
				store.Remove(mockIDs(j, 1)...)
			}
		}(i)
	}

	wg.Wait()

	view, err := store.View()
	assert.NoError(t, err)
	assert.Len(t, view, 0)
}

func TestResultStore_Add(t *testing.T) {
	lggr := log.New(os.Stdout, "", 0)
	store := New(lggr)

	t.Run("happy flow", func(t *testing.T) {
		store.Add(mockItems(0, 10)...)
		assert.Len(t, store.data, 10)
	})

	t.Run("ignore existing items", func(t *testing.T) {
		store.Add(mockItems(0, 10)...)
		store.Add(mockItems(0, 11)...)
		assert.Len(t, store.data, 11)
	})
}

func TestResultStore_View(t *testing.T) {
	lggr := log.New(io.Discard, "", 0)
	store := New(lggr)

	nitems := int32(10)
	store.Add(mockItems(0, int(nitems))...)

	t.Run("no filters", func(t *testing.T) {
		v, err := store.View()
		assert.NoError(t, err)
		assert.Len(t, v, int(atomic.LoadInt32(&nitems)))
	})

	t.Run("filter 1/2 of items with limit of 1/4", func(t *testing.T) {
		i := 0
		limit := int(atomic.LoadInt32(&nitems)) / 4
		v, err := store.View(ocr2keepersv3.WithFilter(func(res ocr2keepers.CheckResult) bool {
			even := i%2 == 0
			i++
			return even
		}), ocr2keepersv3.WithLimit(limit))
		assert.NoError(t, err)
		assert.Len(t, v, limit)
	})

	t.Run("filter all items", func(t *testing.T) {
		v, err := store.View(ocr2keepersv3.WithFilter(func(cr ocr2keepers.CheckResult) bool {
			return false
		}))
		assert.NoError(t, err)
		assert.Len(t, v, 0)
	})

	t.Run("combined filters", func(t *testing.T) {
		i := 0
		beforeLast := int(atomic.LoadInt32(&nitems)) - 1
		v, err := store.View(ocr2keepersv3.WithFilter(func(res ocr2keepers.CheckResult) bool {
			i++
			return i > 6
		}, func(res ocr2keepers.CheckResult) bool {
			return i > beforeLast
		}))
		assert.NoError(t, err)
		assert.Len(t, v, 1)
	})

	t.Run("filter half of items concurrently", func(t *testing.T) {
		workers := 4

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				i := 0
				v, err := store.View(ocr2keepersv3.WithFilter(func(res ocr2keepers.CheckResult) bool {
					even := i%2 == 0
					i++
					return even
				}))
				assert.NoError(t, err)
				assert.Len(t, v, int(atomic.LoadInt32(&nitems))/2)
			}()
		}
		wg.Wait()
	})

	t.Run("sort items by id desc", func(t *testing.T) {
		v, err := store.View(ocr2keepersv3.WithOrder(func(a, b ocr2keepers.CheckResult) bool {
			return a.Payload.ID > b.Payload.ID
		}))
		assert.NoError(t, err)
		assert.Len(t, v, 10)
		assert.Equal(t, "test-id-9", v[0].Payload.ID)
	})

	t.Run("sort items by id desc with limit", func(t *testing.T) {
		v, err := store.View(ocr2keepersv3.WithOrder(func(a, b ocr2keepers.CheckResult) bool {
			return a.Payload.ID > b.Payload.ID
		}), ocr2keepersv3.WithLimit(3))
		assert.NoError(t, err)
		assert.Len(t, v, 3)
		assert.Equal(t, "test-id-9", v[0].Payload.ID)
		assert.Equal(t, "test-id-8", v[1].Payload.ID)
		assert.Equal(t, "test-id-7", v[2].Payload.ID)
	})

	t.Run("ignore expired items", func(t *testing.T) {
		store.lock.Lock()
		el := store.data["test-id-0"]
		el.addedAt = time.Now().Add(-2 * storeTTL)
		store.data["test-id-0"] = el
		store.lock.Unlock()
		v, err := store.View(ocr2keepersv3.WithOrder(func(a, b ocr2keepers.CheckResult) bool {
			return a.Payload.ID < b.Payload.ID
		}), ocr2keepersv3.WithLimit(3))
		assert.NoError(t, err)
		assert.Len(t, v, 3)
		assert.Equal(t, "test-id-1", v[0].Payload.ID)

	})
}

func mockItems(i, count int) []ocr2keepers.CheckResult {
	items := make([]ocr2keepers.CheckResult, count)
	for j := 0; j < count; j++ {
		items[j] = ocr2keepers.CheckResult{
			Retryable: false,
			Payload: ocr2keepers.UpkeepPayload{
				ID: fmt.Sprintf("test-id-%d", i+j),
			},
		}
	}
	return items
}

func mockIDs(i, count int) []string {
	items := make([]string, count)
	for j := 0; j < count; j++ {
		items[j] = fmt.Sprintf("test-id-%d", i+j)
	}
	return items
}
