package util

import (
	"context"
	"fmt"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v2"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrProcessStopped   = fmt.Errorf("worker process has stopped")
	ErrContextCancelled = fmt.Errorf("worker context cancelled")
)

type WorkItemResult struct {
	Worker string
	Data   []ocr2keepers.UpkeepResult
	Err    error
	Time   time.Duration
}

type WorkItem func(context.Context) ([]ocr2keepers.UpkeepResult, error)

type worker struct {
	Name  string
	Queue chan *worker
}

func (w *worker) Do(ctx context.Context, r func(WorkItemResult), wrk WorkItem) {
	start := time.Now()

	var data []ocr2keepers.UpkeepResult
	var err error

	if ctx.Err() != nil {
		err = ctx.Err()
	} else {
		data, err = wrk(ctx)
	}

	r(WorkItemResult{
		Worker: w.Name,
		Data:   data,
		Err:    err,
		Time:   time.Since(start),
	})

	// put itself back on the queue when done
	select {
	case w.Queue <- w:
	default:
	}
}

type GroupedItem struct {
	Group int
	Item  WorkItem
}

type WorkerGroup struct {
	maxWorkers    int
	activeWorkers int
	workers       chan *worker

	queue         *Queue
	input         chan GroupedItem
	chInputNotify chan struct{}
	mu            sync.Mutex
	resultData    map[int][]WorkItemResult
	resultNotify  map[int]chan struct{}

	// channels used to stop processing
	chStopInputs     chan struct{}
	chStopProcessing chan struct{}
	queueClosed      atomic.Bool

	// service state management
	svcCtx    context.Context
	svcCancel context.CancelFunc
	once      sync.Once
}

func NewWorkerGroup(workers int, queue int) *WorkerGroup {
	svcCtx, svcCancel := context.WithCancel(context.Background())
	wg := &WorkerGroup{
		maxWorkers:       workers,
		workers:          make(chan *worker, workers),
		queue:            &Queue{},
		input:            make(chan GroupedItem, 1),
		chInputNotify:    make(chan struct{}, 1),
		resultData:       map[int][]WorkItemResult{},
		resultNotify:     map[int]chan struct{}{},
		chStopInputs:     make(chan struct{}),
		chStopProcessing: make(chan struct{}),
		svcCtx:           svcCtx,
		svcCancel:        svcCancel,
	}

	go wg.run()

	return wg
}

// Do adds a new work item onto the work queue. This function blocks until
// the work queue clears up or the context is cancelled.
func (wg *WorkerGroup) Do(ctx context.Context, w WorkItem, group int) error {

	if ctx.Err() != nil {
		return fmt.Errorf("%w; work not added to queue", ErrContextCancelled)
	}

	if wg.queueClosed.Load() {
		return fmt.Errorf("%w; work not added to queue", ErrProcessStopped)
	}

	gi := GroupedItem{
		Group: group,
		Item:  w,
	}

	wg.mu.Lock()
	if _, ok := wg.resultData[group]; !ok {
		wg.resultData[group] = make([]WorkItemResult, 0)
	}

	if _, ok := wg.resultNotify[group]; !ok {
		wg.resultNotify[group] = make(chan struct{}, 1)
	}
	wg.mu.Unlock()

	select {
	case wg.input <- gi:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w; work not added to queue", ErrContextCancelled)
	case <-wg.svcCtx.Done():
		return fmt.Errorf("%w; work not added to queue", ErrProcessStopped)
	}
}

func (wg *WorkerGroup) NotifyResult(group int) <-chan struct{} {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	ch, ok := wg.resultNotify[group]
	if !ok {
		// if a channel isn't found for the group, create it
		wg.resultNotify[group] = make(chan struct{}, 1)

		return wg.resultNotify[group]
	}

	return ch
}

func (wg *WorkerGroup) Results(group int) []WorkItemResult {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	resultData, ok := wg.resultData[group]
	if !ok {
		wg.resultData[group] = []WorkItemResult{}

		return wg.resultData[group]
	}

	wg.resultData[group] = []WorkItemResult{}

	// results are stored as latest first
	// switch the order to provide oldest first
	if len(resultData) > 1 {
		for i, j := 0, len(resultData)-1; i < j; i, j = i+1, j-1 {
			resultData[i], resultData[j] = resultData[j], resultData[i]
		}
	}

	return resultData
}

func (wg *WorkerGroup) RemoveGroup(group int) {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	delete(wg.resultData, group)
	delete(wg.resultNotify, group)
}

func (wg *WorkerGroup) Stop() {
	wg.once.Do(func() {
		wg.svcCancel()
		wg.queueClosed.Store(true)
		wg.chStopInputs <- struct{}{}
	})
}

func (wg *WorkerGroup) processQueue() {
	for {
		if wg.queue.Len() == 0 {
			break
		}

		value, err := wg.queue.Pop()

		// an error from pop means there is nothing to pop
		// the length check above should protect from that, but just in case
		// this error also breaks the loop
		if err != nil {
			break
		}

		wg.doJob(value)
	}
}

func (wg *WorkerGroup) runQueuing() {
	for {
		select {
		case item := <-wg.input:
			wg.queue.Add(item)

			// notify that new work item came in
			// drop if notification channel is full
			select {
			case wg.chInputNotify <- struct{}{}:
			default:
			}
		case <-wg.chStopInputs:
			wg.chStopProcessing <- struct{}{}
			return
		}
	}
}

func (wg *WorkerGroup) runProcessing() {
	for {
		select {
		// watch notification channel and begin processing queue
		// when notification occurs
		case <-wg.chInputNotify:
			wg.processQueue()
		case <-wg.chStopProcessing:
			return
		}
	}
}

func (wg *WorkerGroup) run() {
	// start listening on the input channel for new jobs
	go wg.runQueuing()

	// main run loop for queued jobs
	wg.runProcessing()

	// run the job queue one more time just in case some
	// new work items snuck in
	wg.processQueue()
}

func (wg *WorkerGroup) doJob(item GroupedItem) {
	var wkr *worker

	// no read or write locks on activeWorkers or maxWorkers because it's
	// assumed the job loop is a single process reading from the job queue
	if wg.activeWorkers < wg.maxWorkers {
		// create a new worker
		wkr = &worker{
			Name:  fmt.Sprintf("worker-%d", wg.activeWorkers+1),
			Queue: wg.workers,
		}
		wg.activeWorkers++
	} else {
		// wait for a worker to be available
		wkr = <-wg.workers
	}

	// have worker do the work
	go wkr.Do(wg.svcCtx, wg.storeResult(item.Group), item.Item)
}

func (wg *WorkerGroup) storeResult(group int) func(result WorkItemResult) {
	return func(result WorkItemResult) {
		wg.mu.Lock()
		defer wg.mu.Unlock()

		_, ok := wg.resultData[group]
		if !ok {
			wg.resultData[group] = make([]WorkItemResult, 0)
		}

		_, ok = wg.resultNotify[group]
		if !ok {
			wg.resultNotify[group] = make(chan struct{}, 1)
		}

		wg.resultData[group] = append([]WorkItemResult{result}, wg.resultData[group]...)

		select {
		case wg.resultNotify[group] <- struct{}{}:
		default:
		}
	}
}

type JobFunc func(context.Context, []ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error)
type JobResultFunc func([]ocr2keepers.UpkeepResult, error)

func RunJobs(ctx context.Context, wg *WorkerGroup, jobs [][]ocr2keepers.UpkeepKey, jobFunc JobFunc, resFunc JobResultFunc) {
	var wait sync.WaitGroup
	end := make(chan struct{}, 1)

	group := rand.Intn(1_000_000_000)

	go func(g *WorkerGroup, w *sync.WaitGroup, ch chan struct{}) {
		for {
			select {
			case <-g.NotifyResult(group):
				//fmt.Println("NotifyResult")
				for _, r := range g.Results(group) {
					resFunc(r.Data, r.Err)
					w.Done()
				}
			case <-ch:
				return
			}
		}
	}(wg, &wait, end)

	for _, job := range jobs {
		wait.Add(1)

		if err := wg.Do(ctx, makeJobFunc(ctx, job, jobFunc), group); err != nil {
			// the makeJobFunc will exit early if the context passed to it has
			// already completed or if the worker process has been stopped
			wait.Done()
			break
		}
	}

	// wait for all results to be read
	wait.Wait()

	// clean up run group resources
	wg.RemoveGroup(group)

	// close the results reader process to clean up resources
	close(end)
}

func makeJobFunc(jobCtx context.Context, value []ocr2keepers.UpkeepKey, jobFunc JobFunc) WorkItem {
	return func(svcCtx context.Context) ([]ocr2keepers.UpkeepResult, error) {
		// the jobFunc should exit in the case that either the job context
		// cancels or the worker service context cancels. To ensure we don't end
		// up with memory leaks, cancel the merged context to release resources.
		ctx, cancel := MergeContextsWithCancel(svcCtx, jobCtx)
		defer cancel()
		return jobFunc(ctx, value)
	}
}

type Queue struct {
	mu     sync.RWMutex
	values []GroupedItem
}

func (q *Queue) Add(values ...GroupedItem) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.values = append(q.values, values...)
}

func (q *Queue) Pop() (GroupedItem, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.values) == 0 {
		return GroupedItem{}, fmt.Errorf("no values to return")
	}

	val := q.values[0]

	if len(q.values) > 1 {
		q.values = q.values[1:]
	} else {
		q.values = []GroupedItem{}
	}

	return val, nil
}

func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.values)
}
