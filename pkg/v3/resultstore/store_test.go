package resultstore

import (
	"context"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
)

var (
	result1 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{1}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 1,
			BlockHash:   [32]byte{1},
		},
	}
	result2 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{2}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 2,
			BlockHash:   [32]byte{2},
		},
	}
	result3 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{3}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 3,
			BlockHash:   [32]byte{3},
		},
	}
	result4 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{4}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 4,
			BlockHash:   [32]byte{4},
		},
	}
	result5 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{5}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 5,
			BlockHash:   [32]byte{5},
		},
	}

	workID1, _ = UpkeepWorkID(result1.UpkeepID.BigInt(), result1.Trigger)
	workID2, _ = UpkeepWorkID(result2.UpkeepID.BigInt(), result2.Trigger)
	workID3, _ = UpkeepWorkID(result3.UpkeepID.BigInt(), result3.Trigger)
	workID4, _ = UpkeepWorkID(result4.UpkeepID.BigInt(), result4.Trigger)
	workID5, _ = UpkeepWorkID(result5.UpkeepID.BigInt(), result5.Trigger)
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
			name: "happy path",
			itemsToAdd: []ocr2keepers.CheckResult{
				result1,
				result2,
				result3,
				result4,
				result5,
			},
			itemsToRemove: []string{workID1, workID2, workID3},
			expected: []ocr2keepers.CheckResult{
				result4,
				result5,
			},
		},
		{
			name: "remove non-existent item",
			itemsToAdd: []ocr2keepers.CheckResult{
				result1,
			},
			itemsToRemove: []string{"boo"},
			expected: []ocr2keepers.CheckResult{
				result1,
			},
		},
		{
			name:       "no items",
			itemsToAdd: []ocr2keepers.CheckResult{},
			itemsToRemove: []string{
				workID4,
				workID5,
			},
			expected: []ocr2keepers.CheckResult{},
		},
		{
			name: "no items to remove",
			itemsToAdd: []ocr2keepers.CheckResult{
				result4,
				result5,
			},
			itemsToRemove: []string{},
			expected: []ocr2keepers.CheckResult{
				result4,
				result5,
			},
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

	store.Add(result1, result2)

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

	store.Remove(workID1, workID2)

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
	store.Add(result1, result2)
	view, err := store.View()
	assert.NoError(t, err)
	assert.Len(t, view, 2)

	<-time.After(gcInterval * 2)

	view, err = store.View()
	assert.NoError(t, err)
	assert.Len(t, view, 0)
}

//func TestResultStore_Concurrency(t *testing.T) {
//	lggr := log.New(io.Discard, "", 0)
//	store := New(lggr)
//
//	workers := 4
//	nitems := int32(1000)
//
//	var wg sync.WaitGroup
//
//	for i := 0; i < workers; i++ {
//		doneWrite := make(chan bool)
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			items := int(atomic.LoadInt32(&nitems))
//			n := items * (i + 1)
//			for j := items * i; j < n; j++ {
//				store.Add(mockItems(j, 1)...)
//			}
//			doneWrite <- true
//		}(i)
//
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			<-doneWrite
//			items := int(atomic.LoadInt32(&nitems))
//			n := items * (i + 1)
//			for j := items * i; j < n; j++ {
//				store.Remove(mockIDs(j, 1)...)
//			}
//		}(i)
//	}
//
//	wg.Wait()
//
//	view, err := store.View()
//	assert.NoError(t, err)
//	assert.Len(t, view, 0)
//}

func TestResultStore_Add(t *testing.T) {
	lggr := log.New(os.Stdout, "", 0)
	store := New(lggr)

	t.Run("happy flow", func(t *testing.T) {
		store.Add(result1, result2)
		assert.Len(t, store.data, 2)
	})

	t.Run("ignore existing items", func(t *testing.T) {
		store.Add(result1, result2)
		store.Add(result1, result3)
		assert.Len(t, store.data, 3)
	})
}

func TestResultStore_View(t *testing.T) {
	lggr := log.New(io.Discard, "", 0)
	store := New(lggr)

	nitems := int32(4)
	store.Add(result1, result2, result3, result4)

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
			return i > 2
		}, func(res ocr2keepers.CheckResult) bool {
			return i > beforeLast
		}))
		assert.NoError(t, err)
		assert.Len(t, v, 1)
	})

	t.Run("filter half of items concurrently", func(t *testing.T) {
		workers := 2

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
			aWorkID, err := UpkeepWorkID(a.UpkeepID.BigInt(), a.Trigger)
			assert.NoError(t, err)
			bWorkID, err := UpkeepWorkID(b.UpkeepID.BigInt(), b.Trigger)
			assert.NoError(t, err)
			return aWorkID > bWorkID
		}))
		assert.NoError(t, err)
		assert.Len(t, v, 4)

		workID, err := UpkeepWorkID(v[0].UpkeepID.BigInt(), v[0].Trigger)
		assert.NoError(t, err)
		assert.Equal(t, "e2e5a4857befdd80f630d4e8dd93a98df8e8c97cb103f9d0de44470aad44619b", workID)
	})

	t.Run("sort items by id desc with limit", func(t *testing.T) {
		v, err := store.View(ocr2keepersv3.WithOrder(func(a, b ocr2keepers.CheckResult) bool {
			aWorkID, err := UpkeepWorkID(a.UpkeepID.BigInt(), a.Trigger)
			assert.NoError(t, err)
			bWorkID, err := UpkeepWorkID(b.UpkeepID.BigInt(), b.Trigger)
			assert.NoError(t, err)
			return aWorkID > bWorkID
		}), ocr2keepersv3.WithLimit(3))
		assert.NoError(t, err)
		assert.Len(t, v, 3)

		workID0, err := UpkeepWorkID(v[0].UpkeepID.BigInt(), v[0].Trigger)
		assert.NoError(t, err)

		workID1, err := UpkeepWorkID(v[1].UpkeepID.BigInt(), v[1].Trigger)
		assert.NoError(t, err)

		workID2, err := UpkeepWorkID(v[2].UpkeepID.BigInt(), v[2].Trigger)
		assert.NoError(t, err)

		assert.Equal(t, "e2e5a4857befdd80f630d4e8dd93a98df8e8c97cb103f9d0de44470aad44619b", workID0)
		assert.Equal(t, "e07969fc7c14b4453a2d8a87c506c95d417d0c9db59d51920bd0c250330103f8", workID1)
		assert.Equal(t, "ae245c333464f3658fc93bc193a39dafbb5be5900c7e6c2eb795f3e271e57079", workID2)
	})

	t.Run("ignore expired items", func(t *testing.T) {
		store.lock.Lock()
		el := store.data["test-id-0"]
		el.addedAt = time.Now().Add(-2 * storeTTL)
		store.data["test-id-0"] = el
		store.lock.Unlock()
		v, err := store.View(ocr2keepersv3.WithOrder(func(a, b ocr2keepers.CheckResult) bool {
			aWorkID, err := UpkeepWorkID(a.UpkeepID.BigInt(), a.Trigger)
			assert.NoError(t, err)
			bWorkID, err := UpkeepWorkID(b.UpkeepID.BigInt(), b.Trigger)
			assert.NoError(t, err)
			return aWorkID < bWorkID
		}), ocr2keepersv3.WithLimit(3))
		assert.NoError(t, err)
		assert.Len(t, v, 3)

		workID0, err := UpkeepWorkID(v[0].UpkeepID.BigInt(), v[0].Trigger)
		assert.NoError(t, err)

		assert.Equal(t, "65904d6a23823ee6cf86fab18a883201622f39ab6e9eb07dfd0902f83a9aff85", workID0)

	})
}

//
//func mockItems(i, count int) []ocr2keepers.CheckResult {
//	items := make([]ocr2keepers.CheckResult, count)
//	for j := 0; j < count; j++ {
//		items[j] = ocr2keepers.CheckResult{
//			Retryable: false,
//		}
//	}
//	return items
//}
//
//func mockIDs(i, count int) []string {
//	items := make([]string, count)
//
//	for j := 0; j < count; j++ {
//		items[j] = fmt.Sprintf("testid%d", i+j)
//	}
//
//	return items
//}
