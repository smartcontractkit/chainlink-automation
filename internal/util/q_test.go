package util

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	t.Run("Push", func(t *testing.T) {
		q := NewQueue[string]()
		q.Push("a", "b", "c")
		require.Equal(t, 3, q.Size())
	})

	t.Run("Pop", func(t *testing.T) {
		q := NewQueue[string]()
		added := []string{"a", "b", "c"}
		q.Push(added...)
		require.Equal(t, 3, q.Size())
		require.Len(t, q.Pop(1), 1, "should pop 1 item")
		require.Len(t, q.Pop(-1), len(added)-1, "should pop remaining items")
		require.Equal(t, 0, q.Size())
		require.Len(t, q.Pop(1), 0, "empty queue shouldn't have items")
		require.Len(t, q.Pop(0), 0, "empty queue shouldn't have items")
	})

	t.Run("PopF", func(t *testing.T) {
		q := NewQueue[string]()
		added := []string{"a", "b", "c"}
		q.Push(added...)

		require.Equal(t, 3, q.Size())
		require.Len(t, q.PopF(func(s string) bool {
			return s == "a"
		}), 1, "should pop a single item")
		require.Equal(t, 2, q.Size())
		require.Len(t, q.PopF(func(s string) bool {
			return true
		}), len(added)-1, "should pop the remaining items")
		require.Equal(t, 0, q.Size())
		require.Len(t, q.PopF(func(s string) bool {
			return true
		}), 0, "empty queue shouldn't have items")
	})
}

func TestQueue_Concurrency(t *testing.T) {
	pctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := NewQueue[string]()

	go func() {
		q.Push("a", "b", "c")
	}()
	go func() {
		q.Push("d", "e", "f")
	}()

	acc := []string{}
	for pctx.Err() == nil {
		popped := q.Pop(-1)
		if len(popped) > 0 {
			acc = append(acc, popped...)
		}
		if len(acc) == 6 {
			break
		}
		runtime.Gosched()
	}
	require.Equal(t, 6, len(acc))
}
