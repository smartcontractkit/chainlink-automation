package resultstore

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type checkResult struct {
	Retryable bool
	Data      string
}

func TestResultStore_Sanity(t *testing.T) {
	lggr := log.New(os.Stdout, "", 0)
	key := func(t checkResult) string {
		return t.Data
	}

	tests := []struct {
		name          string
		itemsToAdd    []checkResult
		itemsToRemove []checkResult
		expected      []checkResult
	}{
		{
			name: "happy path 5 items",
			itemsToAdd: []checkResult{
				{
					Retryable: false,
					Data:      "some data 1",
				},
				{
					Retryable: false,
					Data:      "some data 2",
				},
				{
					Retryable: false,
					Data:      "some data 3",
				},
				{
					Retryable: false,
					Data:      "some data 4",
				},
				{
					Retryable: false,
					Data:      "some data 5",
				},
			},
			itemsToRemove: []checkResult{
				{
					Retryable: false,
					Data:      "some data 1",
				},
				{
					Retryable: false,
					Data:      "some data 3",
				},
				{
					Retryable: false,
					Data:      "some data 5",
				},
			},
			expected: []checkResult{
				{
					Retryable: false,
					Data:      "some data 2",
				},
				{
					Retryable: false,
					Data:      "some data 4",
				},
			},
		},
		{
			name: "remove non-existent item",
			itemsToAdd: []checkResult{
				{
					Retryable: false,
					Data:      "some data 1",
				},
			},
			itemsToRemove: []checkResult{
				{
					Retryable: false,
					Data:      "some data 2",
				},
			},
			expected: []checkResult{
				{
					Retryable: false,
					Data:      "some data 1",
				},
			},
		},
		{
			name:          "no items",
			itemsToAdd:    []checkResult{},
			itemsToRemove: []checkResult{},
			expected:      []checkResult{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := NewResultStore(lggr, key)
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
			for _, v := range view {
				assert.Contains(t, tc.expected, v)
				assert.NotContains(t, tc.itemsToRemove, v)
			}
		})
	}

}
