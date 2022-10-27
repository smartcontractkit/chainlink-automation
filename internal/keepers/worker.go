package keepers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

var (
	ErrProcessStopped   = fmt.Errorf("worker process has stopped")
	ErrContextCancelled = fmt.Errorf("worker context cancelled")
	ErrQueueFull        = fmt.Errorf("worker queue full")
)

type workResult[T any] struct {
	Worker string
	Data   T
	Err    error
	Time   time.Duration
}

type work[T any] func(context.Context) (T, error)

type worker[T any] struct {
	Name  string
	Queue chan *worker[T]
}

func (w *worker[T]) Do(ctx context.Context, r chan workResult[T], wrk work[T]) {
	start := time.Now()

	data, err := wrk(ctx)
	result := workResult[T]{
		Worker: w.Name,
		Data:   data,
		Err:    err,
		Time:   time.Since(start),
	}

	select {
	case r <- result:
		// put itself back on the queue when done
		select {
		case w.Queue <- w:
			return
		case <-ctx.Done():
			return
		}
	case <-ctx.Done():
		return
	}
}

type workerGroup[T any] struct {
	maxWorkers    int
	activeWorkers int
	workers       chan *worker[T]
	queue         chan work[T]
	queueClosed   bool
	stop          chan struct{}
	results       chan workResult[T]
	mu            sync.Mutex
}

func newWorkerGroup[T any](workers int, queue int) *workerGroup[T] {
	wg := &workerGroup[T]{
		maxWorkers: workers,
		workers:    make(chan *worker[T], workers),
		queue:      make(chan work[T], queue),
		stop:       make(chan struct{}, 1),
		results:    make(chan workResult[T], queue),
	}

	go func(g *workerGroup[T]) {
		// timer := time.NewTimer(5 * time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		for {
			select {
			case item := <-g.queue:
				var wkr *worker[T]
				if g.activeWorkers < g.maxWorkers {
					// create a new worker
					wkr = &worker[T]{
						Name:  fmt.Sprintf("worker-%d", g.activeWorkers+1),
						Queue: g.workers,
					}
					g.activeWorkers++
				} else {
					// wait for a worker to be available
					wkr = <-g.workers
				}

				// have worker do the work
				go wkr.Do(ctx, g.results, item)

				// timer.Reset(5 * time.Second)
				/*
					case <-timer.C:
						// close workers when not needed
						if g.activeWorkers > 0 {
							<-g.workers
							g.activeWorkers--
						}
				*/
			case <-g.stop:
				g.mu.Lock()
				close(g.queue)
				g.queueClosed = true
				g.mu.Unlock()
				cancel()
				return
			}
		}
	}(wg)

	runtime.SetFinalizer(wg, func(g *workerGroup[T]) { close(g.stop) })

	return wg
}

// Do adds a new work item onto the work queue. This function blocks until
// the work queue clears up or the context is cancelled.
func (wg *workerGroup[T]) Do(ctx context.Context, w work[T]) error {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	if ctx.Err() != nil {
		return fmt.Errorf("%w; work not added to queue", ErrContextCancelled)
	}

	if wg.queueClosed {
		return fmt.Errorf("%w; work not added to queue", ErrProcessStopped)
	}

	select {
	case wg.queue <- w:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w; work not added to queue", ErrContextCancelled)
	case <-wg.stop:
		return fmt.Errorf("%w; work not added to queue", ErrProcessStopped)
	}
}

func (wg *workerGroup[T]) Stop() {
	close(wg.stop)
}
