package stores

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
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

	q := NewRetryQueue(log.New(io.Discard, "", 0))

	err := q.Enqueue(
		newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "1"}, 0),
		newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "2"}, time.Millisecond*5),
	)
	require.NoError(t, err)

	err = q.Enqueue(
		newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "2"}, 0),
		newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "3"}, 0),
	)
	require.NoError(t, err)

	require.Equal(t, 3, q.Size())

	// dequeue before retry interval elapsed
	items, err := q.Dequeue(2)
	require.NoError(t, err)
	require.Len(t, items, 0)

	require.Equal(t, 3, q.Size())

	// adding a record with a custom interval
	err = q.Enqueue(
		newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "4"}, defaultExpiration-time.Millisecond*5),
	)
	require.NoError(t, err)
	require.Equal(t, 4, q.Size())
	// dequeue after retry interval elapsed
	go func() {
		defer cancel()
		<-time.After(retryInterval * 2)
		items, err = q.Dequeue(2)
		require.NoError(t, err)
		require.Len(t, items, 2)

		require.Equal(t, 2, q.Size())
	}()

	<-ctx.Done()
}

func TestRetryQueue_Expiration(t *testing.T) {
	defaultExpiration := time.Second / 10
	retryInterval := time.Millisecond * 10
	revert := overrideDefaults(defaultExpiration, retryInterval)
	defer revert()

	q := NewRetryQueue(log.New(io.Discard, "", 0))

	t.Run("dequeue before expiration", func(t *testing.T) {
		err := q.Enqueue(
			newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "1"}, 0),
			newRetryRecord(ocr2keepers.UpkeepPayload{WorkID: "2"}, 0),
		)
		require.NoError(t, err)
		require.Equal(t, 2, q.Size())
		items, err := q.Dequeue(20)
		require.NoError(t, err)
		require.Len(t, items, 0)
	})

	t.Run("dequeue after expiration", func(t *testing.T) {
		<-time.After(defaultExpiration * 2)
		require.Equal(t, 0, q.Size())
		q.lock.RLock()
		n := len(q.records)
		q.lock.RUnlock()
		require.Equal(t, 2, n)
		items, err := q.Dequeue(2)
		require.NoError(t, err)
		require.Len(t, items, 0)
		// ensure all expired payloads were removed during dequeue
		q.lock.RLock()
		n = len(q.records)
		q.lock.RUnlock()
		require.Equal(t, 0, n)
	})
}

func newRetryRecord(payload ocr2keepers.UpkeepPayload, interval time.Duration) ocr2keepers.RetryRecord {
	return ocr2keepers.RetryRecord{
		Payload:  payload,
		Interval: interval,
	}
}
