package ocr2keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type checkResult struct {
	Retryable bool
	Data      string
}

func TestNewResultStore(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		store := NewResultStore[checkResult]()
		result1 := checkResult{
			Retryable: false,
			Data:      "some data",
		}

		store.Add(result1)
		results, err := store.View()
		assert.Nil(t, err)
		assert.Len(t, results, 1)

		store.Add(result1)
		results, err = store.View()
		assert.Nil(t, err)
		assert.Len(t, results, 1)

		result2 := checkResult{
			Retryable: false,
			Data:      "some data",
		}
		store.Add(result2)
		results, err = store.View()
		assert.Nil(t, err)
		assert.Len(t, results, 1)

		result3 := checkResult{
			Retryable: false,
			Data:      "some other data",
		}
		store.Add(result3)
		results, err = store.View()
		assert.Nil(t, err)
		assert.Len(t, results, 2)

		store.Add(result1, result2, result3)
		results, err = store.View()
		assert.Nil(t, err)
		assert.Len(t, results, 2)

		t.Run("remove", func(t *testing.T) {
			store.Remove(result2)
			results, err = store.View()
			assert.Nil(t, err)
			assert.Len(t, results, 1)

			store.Remove(result2)
			results, err = store.View()
			assert.Nil(t, err)
			assert.Len(t, results, 1)

			store.Remove(result2, result2)
			results, err = store.View()
			assert.Nil(t, err)
			assert.Len(t, results, 1)

			store.Remove(result3)
			results, err = store.View()
			assert.Nil(t, err)
			assert.Len(t, results, 0)
		})
	})

}
