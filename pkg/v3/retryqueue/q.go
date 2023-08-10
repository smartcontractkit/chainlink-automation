package retryqueue

import (
	"fmt"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type RetryQueue interface {
	// Enqueue adds new items to the queue
	Enqueue(items ...types.UpkeepPayload) error
	// Dequeue returns the next n items in the queue, considering retry time schedules
	Dequeue(n int) ([]types.UpkeepPayload, error)
}

var (
	DefaultExpiration = 24 * time.Hour
	RetryInterval     = 5 * time.Minute

	ErrNonRetryable = fmt.Errorf("item is not retryable")
)

type retryQueueItem struct {
	payload types.UpkeepPayload
	// dequeued is true if the item is currently being retried,
	// indicating that it should not be retried at the moment.
	dequeued bool
	// firstRetry is the time the item was first added to the queue
	firstRetry time.Time
	// lastRetry is the last time the item was sent to the queue
	lastRetry time.Time
}

type retryQueue struct {
	payloads map[string]retryQueueItem
	lock     sync.Mutex
}

var _ RetryQueue = (*retryQueue)(nil)

func NewRetryQueue() *retryQueue {
	return &retryQueue{
		payloads: map[string]retryQueueItem{},
		lock:     sync.Mutex{},
	}
}

func (q *retryQueue) Enqueue(payloads ...types.UpkeepPayload) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	now := time.Now()

	for _, payload := range payloads {
		item, ok := q.payloads[payload.WorkID]
		if !ok {
			item = retryQueueItem{
				payload:    payload,
				firstRetry: now,
			}
		}
		if payload.Trigger.BlockNumber > item.payload.Trigger.BlockNumber {
			// new item is newer -> replace
			item.payload = payload
		}
		item.lastRetry = now
		item.dequeued = false
		q.payloads[payload.WorkID] = item
	}

	return nil
}

func (q *retryQueue) Dequeue(n int) ([]types.UpkeepPayload, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	now := time.Now()

	var results []types.UpkeepPayload
	for k, item := range q.payloads {
		if now.Sub(item.firstRetry) > DefaultExpiration {
			// expired -> remove from queue
			delete(q.payloads, k)
			continue
		}
		if item.dequeued {
			continue
		}
		if now.Sub(item.lastRetry) > RetryInterval {
			// within retry interval/window -> add to results and mark as dequeued
			results = append(results, item.payload)
			item.dequeued = true
			q.payloads[k] = item
			if len(results) >= n {
				break
			}
		}
	}

	return results, nil
}

// Size returns the number of items in the queue that are not expired
func (q *retryQueue) Size() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	now := time.Now()
	size := 0

	for _, item := range q.payloads {
		if item.dequeued || now.Sub(item.firstRetry) > DefaultExpiration {
			continue
		}
		size++
	}

	return size
}
