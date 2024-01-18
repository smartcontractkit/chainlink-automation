package stores

import (
	"context"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/stretchr/testify/assert"
)

var (
	result1 = ocr2keepers.CheckResult{
		Retryable: false,
		WorkID:    "workID1",
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{1}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 1,
			BlockHash:   [32]byte{1},
		},
	}
	result2 = ocr2keepers.CheckResult{
		Retryable: false,
		WorkID:    "workID2",
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{2}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 2,
			BlockHash:   [32]byte{2},
		},
	}
	result3 = ocr2keepers.CheckResult{
		Retryable: false,
		WorkID:    "workID3",
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{3}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 3,
			BlockHash:   [32]byte{3},
		},
	}
	result4 = ocr2keepers.CheckResult{
		Retryable: false,
		WorkID:    "workID4",
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{4}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 4,
			BlockHash:   [32]byte{4},
		},
	}
	result5 = ocr2keepers.CheckResult{
		Retryable: false,
		WorkID:    "workID5",
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{5}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 5,
			BlockHash:   [32]byte{5},
		},
	}
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
			itemsToRemove: []string{"workID1", "workID2", "workID3"},
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
				"workID4",
				"workID5",
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
	lggr := log.New(io.Discard, "", 0)
	store := New(lggr)

	store.Add(result1, result2)
	var wg sync.WaitGroup

	store.Remove("workID1", "workID2")

	store.lock.Lock()
	el := store.data["test-id-2"]
	el.addedAt = time.Now().Add(-2 * storeTTL)
	store.data["test-id-2"] = el
	store.lock.Unlock()

	store.gc()

	wg.Wait()
}

func TestResultStore_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lggr := log.New(io.Discard, "", 0)

	store := New(lggr)
	origGcInterval := gcInterval
	origStoreTTL := storeTTL

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

	store.Close()

	<-store.closedCh
	gcInterval = origGcInterval
	storeTTL = origStoreTTL
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

	t.Run("no filters", func(t *testing.T) {
		nitems := int32(4)
		store := New(lggr)
		store.Add(result1, result2, result3, result4)
		v, err := store.View()
		assert.NoError(t, err)
		assert.Len(t, v, int(atomic.LoadInt32(&nitems)))
	})

	t.Run("ignore expired items", func(t *testing.T) {
		store := New(lggr)
		store.Add(result1, result2)
		store.lock.Lock()
		el := store.data["workID1"]
		el.addedAt = time.Now().Add(-2 * storeTTL)
		store.data["workID1"] = el
		store.lock.Unlock()
		v, err := store.View()
		assert.NoError(t, err)
		assert.Len(t, v, 1)

		assert.Equal(t, "workID2", v[0].WorkID)

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
