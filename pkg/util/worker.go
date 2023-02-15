package util

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
)

var (
	ErrProcessStopped   = fmt.Errorf("worker process has stopped")
	ErrContextCancelled = fmt.Errorf("worker context cancelled")
)

type WorkItemResult[T any] struct {
	Worker string
	Data   T
	Err    error
	Time   time.Duration
}

type WorkItem[T any] func(context.Context) (T, error)

type worker[T any] struct {
	Name  string
	Queue chan *worker[T]
}

func (w *worker[T]) Do(ctx context.Context, r chan WorkItemResult[T], wrk WorkItem[T]) {
	start := time.Now()

	data, err := wrk(ctx)
	result := WorkItemResult[T]{
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

type WorkerGroup[T any] struct {
	maxWorkers    int
	activeWorkers int
	workers       chan *worker[T]
	queue         chan WorkItem[T]
	queueClosed   bool
	stop          chan struct{}
	Results       chan WorkItemResult[T]
	mu            sync.Mutex
	once          sync.Once
}

func NewWorkerGroup[T any](workers int, queue int) *WorkerGroup[T] {
	wg := &WorkerGroup[T]{
		maxWorkers: workers,
		workers:    make(chan *worker[T], workers),
		queue:      make(chan WorkItem[T], queue),
		stop:       make(chan struct{}, 1),
		Results:    make(chan WorkItemResult[T], queue),
	}

	go func(g *WorkerGroup[T]) {
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
				go wkr.Do(ctx, g.Results, item)

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
				if !g.queueClosed {
					close(g.queue)
					g.queueClosed = true
				}
				g.mu.Unlock()
				cancel()
				return
			}
		}
	}(wg)

	runtime.SetFinalizer(wg, func(g *WorkerGroup[T]) { g.Stop() })

	return wg
}

// Do adds a new work item onto the work queue. This function blocks until
// the work queue clears up or the context is cancelled.
func (wg *WorkerGroup[T]) Do(ctx context.Context, w WorkItem[T]) error {
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

func (wg *WorkerGroup[T]) Stop() {
	wg.once.Do(func() {
		close(wg.stop)
	})
}

type JobFunc[T, K any] func(context.Context, T) (K, error)
type JobResultFunc[T any] func(T, error)

func RunJobs[T, K any](ctx context.Context, wg *WorkerGroup[T], jobs []K, jobFunc JobFunc[K, T], resFunc JobResultFunc[T]) {
	var wait sync.WaitGroup
	end := make(chan struct{})

	go func(g *WorkerGroup[T], w *sync.WaitGroup, ch chan struct{}) {
		for {
			select {
			case r := <-g.Results:
				resFunc(r.Data, r.Err)
				w.Done()
			case <-ch:
				return
			}
		}
	}(wg, &wait, end)

	for _, job := range jobs {
		wait.Add(1)

		if err := wg.Do(ctx, makeJobFunc(ctx, job, jobFunc)); err != nil {
			if !errors.Is(err, ErrContextCancelled) {
				// the worker group process has probably stopped so the job runner
				// should close outstanding resources and terminate
				close(end)
				return
			}

			// the makeJobFunc will exit early if the context passed to it has
			// already completed.
			wait.Done()
		}
	}

	wait.Wait()
	close(end)
}

func makeJobFunc[T, K any](jobCtx context.Context, value T, jobFunc JobFunc[T, K]) WorkItem[K] {
	return func(svcCtx context.Context) (K, error) {
		// the jobFunc should exit in the case that either the job context
		// cancels or the worker service context cancels. To ensure we don't end
		// up with memory leaks, cancel the merged context to release resources.
		ctx, cancel := MergeContextsWithCancel(svcCtx, jobCtx)
		defer cancel()
		return jobFunc(ctx, value)
	}
}
