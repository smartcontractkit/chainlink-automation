package util

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncedArray(t *testing.T) {
	t.Run("simple append operations update the array as expected", func(t *testing.T) {
		s := NewSyncedArray[int]()

		s.Append(1, 2, 3)

		assert.Equal(t, []int{1, 2, 3}, s.Values())

		s.Append(4, 5, 6)

		assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, s.Values())
	})

	t.Run("parallel append operations update the array as expected", func(t *testing.T) {
		s := NewSyncedArray[int]()

		var wg sync.WaitGroup
		wg.Add(6)

		go func() {
			s.Append(5, 4, 3, 2)
			wg.Done()
		}()
		go func() {
			s.Append(1)
			wg.Done()
		}()
		go func() {
			s.Append(999, 999, 111)
			wg.Done()
		}()
		go func() {
			s.Append(7, 6, 5)
			wg.Done()
		}()
		go func() {
			s.Append(9, 1, 0)
			wg.Done()
		}()
		go func() {
			s.Append(4, 8, 2)
			wg.Done()
		}()

		wg.Wait()

		sort.Ints(s.Values())

		assert.Equal(t, []int{0, 1, 1, 2, 2, 3, 4, 4, 5, 5, 6, 7, 8, 9, 111, 999, 999}, s.Values())
	})
}

//
//func TestUnflatten(t *testing.T) {
//	groups := Unflatten[int]([]ocr2keepers.UpkeepKey{0, 1, 1, 2, 2, 3, 4, 4, 5, 5, 6, 7, 8, 9, 111, 999, 999}, 3)
//
//	assert.Equal(t, 6, len(groups))
//	assert.Equal(t, []int{0, 1, 1}, groups[0])
//	assert.Equal(t, []int{2, 2, 3}, groups[1])
//	assert.Equal(t, []int{4, 4, 5}, groups[2])
//	assert.Equal(t, []int{5, 6, 7}, groups[3])
//	assert.Equal(t, []int{8, 9, 111}, groups[4])
//	assert.Equal(t, []int{999, 999}, groups[5])
//}
