package retryqueue

import (
	"context"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	"github.com/stretchr/testify/require"
)

func overrideDefaults(expiration, retryInterval time.Duration) func() {
	origDefaultExpiration := DefaultExpiration
	origRetryInterval := RetryInterval
	DefaultExpiration = expiration
	RetryInterval = retryInterval
	return func() {
		DefaultExpiration = origDefaultExpiration
		RetryInterval = origRetryInterval
	}
}

func TestRetryQueue_Sanity(t *testing.T) {
	defaultExpiration := time.Second * 1
	retryInterval := time.Millisecond * 10
	ctx, cancel := context.WithTimeout(context.Background(), 2*defaultExpiration)
	defer cancel()

	revert := overrideDefaults(defaultExpiration, retryInterval)
	defer revert()

	q := NewRetryQueue()

	err := q.Enqueue(
		ocr2keepers.UpkeepPayload{WorkID: "1"},
		ocr2keepers.UpkeepPayload{WorkID: "2"},
	)
	require.NoError(t, err)

	err = q.Enqueue(
		ocr2keepers.UpkeepPayload{WorkID: "2"},
		ocr2keepers.UpkeepPayload{WorkID: "3"},
	)
	require.NoError(t, err)

	require.Equal(t, 3, q.Size())

	// dequeue before retry interval elapsed
	items, err := q.Dequeue(2)
	require.NoError(t, err)
	require.Len(t, items, 0)

	require.Equal(t, 3, q.Size())
	// dequeue after retry interval elapsed
	go func() {
		defer cancel()
		<-time.After(retryInterval * 2)
		items, err = q.Dequeue(2)
		require.NoError(t, err)
		require.Len(t, items, 2)

		require.Equal(t, 1, q.Size())

		err := q.Enqueue(
			ocr2keepers.UpkeepPayload{WorkID: "1"},
		)
		require.NoError(t, err)
		require.Equal(t, 2, q.Size())
	}()

	<-ctx.Done()
}
