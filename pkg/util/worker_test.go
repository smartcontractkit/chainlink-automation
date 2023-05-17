package util

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	t.Run("Return Complete Result to Result Function", func(t *testing.T) {
		w := &worker[int]{
			Name:  "worker",
			Queue: make(chan *worker[int]), // leave this unbuffered to not allow anything to block the queue
		}

		var c sync.Mutex
		var resultItem *WorkItemResult[int]

		f := func(_ context.Context) (int, error) {
			return 10, fmt.Errorf("error")
		}

		ctx := context.Background()
		w.Do(ctx, func(item WorkItemResult[int]) { c.Lock(); defer c.Unlock(); resultItem = &item }, f)

		// assert that result item is not nil and contains expected data
		assert.Eventually(
			t,
			func() bool {
				c.Lock()
				defer c.Unlock()
				return resultItem != nil &&
					resultItem.Err != nil &&
					10 == resultItem.Data &&
					resultItem.Err.Error() == fmt.Errorf("error").Error()
			},
			20*time.Millisecond,
			5*time.Millisecond,
			"data from work result should match",
		)
	})

	t.Run("Return Context Error on Cancelled Context", func(t *testing.T) {
		w := &worker[int]{
			Name:  "worker",
			Queue: make(chan *worker[int]), // leave this unbuffered to block the queue
		}

		var c sync.Mutex
		var resultItem *WorkItemResult[int]

		f := func(_ context.Context) (int, error) {
			return 10, fmt.Errorf("error")
		}

		ctx, cancel := context.WithCancel(context.Background())

		// immediately cancel the context to simulate providing the following
		// function a cancelled context
		cancel()

		// do not run in go-routine to ensure the function does not block
		w.Do(ctx, func(item WorkItemResult[int]) { c.Lock(); defer c.Unlock(); resultItem = &item }, f)

		assert.NotNil(t, resultItem)
		if resultItem != nil {
			assert.Error(t, resultItem.Err)
			assert.Equal(t, "context canceled", resultItem.Err.Error())
		}
	})
}

func TestWorkerGroup(t *testing.T) {

	t.Run("All Work Done", func(t *testing.T) {
		wg := NewWorkerGroup[bool](8, 1000)
		ctx := context.Background()
		group := 0

		closed := make(chan struct{})
		var done int
		go func(w *WorkerGroup[bool], c context.Context) {
			for {
				tmr := time.NewTimer(50 * time.Millisecond)
				select {
				case <-w.NotifyResult(group):
					tmr.Stop()
					for range w.Results(group) {
						done++
					}
					continue
				case <-tmr.C:
					close(closed)
					return
				}
			}
		}(wg, ctx)

		for n := 0; n < 10; n++ {
			err := wg.Do(ctx, func(c context.Context) (bool, error) {
				return true, nil
			}, group)
			if err != nil {
				t.FailNow()
			}
		}

		<-closed

		assert.Equal(t, 10, done)
	})

	t.Run("Wait Before Sending", func(t *testing.T) {
		// the worker group destroys workers when nothing is in the queue
		// for longer than 1 second. this test ensures that total number of
		// workers doesn't go below 0 causing the worker group to lock.
		wg := NewWorkerGroup[bool](8, 1000)
		ctx := context.Background()
		group := 0

		<-time.After(1500 * time.Millisecond)

		closed := make(chan struct{})
		var done int
		go func(w *WorkerGroup[bool], c context.Context) {
			for {
				tmr := time.NewTimer(50 * time.Millisecond)
				select {
				case <-w.NotifyResult(group):
					tmr.Stop()
					for range w.Results(group) {
						done++
					}
					continue
				case <-tmr.C:
					close(closed)
					return
				}
			}
		}(wg, ctx)

		for n := 0; n < 10; n++ {
			err := wg.Do(ctx, func(c context.Context) (bool, error) {
				return true, nil
			}, group)
			if err != nil {
				t.FailNow()
			}
		}

		<-closed

		assert.Equal(t, 10, done)
	})

	t.Run("Error on Cancel and Full Queue", func(t *testing.T) {
		svcCtx, svcCancel := context.WithCancel(context.Background())
		wg := &WorkerGroup[int]{
			svcCtx:           svcCtx,
			svcCancel:        svcCancel,
			input:            make(chan GroupedItem[int]), // unbuffered to block
			resultData:       map[int][]WorkItemResult[int]{},
			resultNotify:     map[int]chan struct{}{},
			chStopInputs:     make(chan struct{}, 1),
			chStopProcessing: make(chan struct{}, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		errors := make(chan error, 1)
		group := 0

		go func(errs chan error, gp int) {
			err := wg.Do(ctx, func(_ context.Context) (int, error) {
				return 1, nil
			}, gp)

			errs <- err
		}(errors, group)

		// wait for a short period to ensure function is in select statement
		<-time.After(20 * time.Millisecond)
		cancel()

		// read errors to ensure errors are expected
		tmr := time.NewTimer(20 * time.Millisecond)
		select {
		case err := <-errors:
			tmr.Stop()
			assert.ErrorIs(t, err, ErrContextCancelled)
		case <-tmr.C:
			assert.Fail(t, "error expected but not found")
			break
		}
	})

	t.Run("Error on Stop and Full Queue", func(t *testing.T) {
		svcCtx, svcCancel := context.WithCancel(context.Background())
		wg := &WorkerGroup[int]{
			svcCtx:           svcCtx,
			svcCancel:        svcCancel,
			input:            make(chan GroupedItem[int]), // unbuffered to block
			resultData:       map[int][]WorkItemResult[int]{},
			resultNotify:     map[int]chan struct{}{},
			chStopInputs:     make(chan struct{}, 1),
			chStopProcessing: make(chan struct{}, 1),
		}

		errors := make(chan error, 1)
		group := 0

		go func(errs chan error, gp int) {
			err := wg.Do(context.Background(), func(_ context.Context) (int, error) {
				return 1, nil
			}, gp)

			errs <- err
		}(errors, group)

		// wait for a short period to ensure function is in queue
		<-time.After(20 * time.Millisecond)
		wg.Stop()

		// read errors to ensure errors are expected
		tmr := time.NewTimer(20 * time.Millisecond)
		select {
		case err := <-errors:
			tmr.Stop()
			assert.ErrorIs(t, err, ErrProcessStopped)
		case <-tmr.C:
			assert.Fail(t, "error expected but not found")
			break
		}
	})

	t.Run("Error on Context Already Cancelled", func(t *testing.T) {
		svcCtx, svcCancel := context.WithCancel(context.Background())
		wg := &WorkerGroup[int]{
			svcCtx:           svcCtx,
			svcCancel:        svcCancel,
			input:            make(chan GroupedItem[int]), // unbuffered to block
			resultData:       map[int][]WorkItemResult[int]{},
			resultNotify:     map[int]chan struct{}{},
			chStopInputs:     make(chan struct{}, 1),
			chStopProcessing: make(chan struct{}, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		errors := make(chan error, 1)
		group := 0

		cancel()

		go func(errs chan error, gp int) {
			err := wg.Do(ctx, func(_ context.Context) (int, error) {
				return 1, nil
			}, gp)

			errs <- err
		}(errors, group)

		// read errors to ensure errors are expected
		tmr := time.NewTimer(20 * time.Millisecond)
		select {
		case err := <-errors:
			tmr.Stop()
			assert.ErrorIs(t, err, ErrContextCancelled)
		case <-tmr.C:
			assert.Fail(t, "error expected but not found")
			break
		}
	})

	t.Run("Stop Closes Queue and Ends Run Context", func(t *testing.T) {
		wg := NewWorkerGroup[int](1, 1)
		group := 0

		go func(gp int) {
			err := wg.Do(context.Background(), func(ctx context.Context) (int, error) {
				tmr := time.NewTimer(100 * time.Millisecond)
				select {
				case <-ctx.Done():
					tmr.Stop()
					return 0, fmt.Errorf("error")
				case <-tmr.C:
					return 1, nil
				}
			}, gp)

			assert.NoError(t, err, "adding work to group should not return error")
		}(group)

		// give some time for queue to start work
		<-time.After(20 * time.Millisecond)

		// stop process to close the queue and cancel the run context
		wg.Stop()

		// check results to see that work item returned short due to context cancel
		// may not have results, which is expected behavior
		tmr := time.NewTimer(150 * time.Millisecond)
		select {
		case <-wg.NotifyResult(group):
			for _, r := range wg.Results(group) {
				// should have a result
				assert.Equal(t, 0, r.Data, "data from work result should match")
				assert.Equal(t, fmt.Errorf("error"), r.Err, "error from work result should match")
			}
			tmr.Stop()
		case <-tmr.C:
			break
		}

		// check that queue is closed
		testAdd := func() error {
			return wg.Do(context.Background(), func(ctx context.Context) (int, error) {
				return 0, nil
			}, group)
		}

		assert.ErrorIs(t, testAdd(), ErrProcessStopped, "queue should be closed")
	})
}

func TestRunJobs(t *testing.T) {

	var jobFunc = func(ctx context.Context, v uint) (int, error) {
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
			return int(v), nil
		}
	}

	var resultFuncWrapper = func(result *int, errors *int) func(i int, err error) {
		return func(i int, err error) {
			if err == nil {
				o := *result + i
				*result = o
			} else {
				o := *errors + 1
				*errors = o
			}
		}
	}

	t.Run("Run All Jobs To Completion", func(t *testing.T) {
		wg := NewWorkerGroup[int](10, 100)

		jobCount := 100
		jobs := make([]uint, jobCount)
		for i := 0; i < jobCount; i++ {
			jobs[i] = 1
		}

		var result int
		var errors int

		RunJobs(context.Background(), wg, jobs, jobFunc, resultFuncWrapper(&result, &errors))

		wg.Stop()

		assert.Equal(t, jobCount, result)
		assert.Equal(t, 0, errors)
	})

	t.Run("Cancel Jobs Before Complete", func(t *testing.T) {
		wg := NewWorkerGroup[int](10, 100)

		jobCount := 100
		jobs := make([]uint, jobCount)
		for i := 0; i < jobCount; i++ {
			jobs[i] = 1
		}

		var result int
		var errors int

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(200 * time.Millisecond)
			cancel()
		}()

		RunJobs(ctx, wg, jobs, jobFunc, resultFuncWrapper(&result, &errors))

		wg.Stop()

		assert.Greater(t, errors, 0)
		assert.Greater(t, result, 0)
		assert.Less(t, result, jobCount)
		assert.Less(t, errors, jobCount)

		assert.Equal(t, jobCount, result+errors)
	})

	t.Run("Small Queue w/ Cancel", func(t *testing.T) {
		// make the queue size much smaller than the job count
		wg := NewWorkerGroup[int](10, 10)

		jobCount := 100
		jobs := make([]uint, jobCount)
		for i := 0; i < jobCount; i++ {
			jobs[i] = 1
		}

		var result int
		var errors int
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.After(200 * time.Millisecond)
			cancel()
		}()

		RunJobs(ctx, wg, jobs, jobFunc, resultFuncWrapper(&result, &errors))

		wg.Stop()

		assert.Greater(t, errors, 0)
		assert.Greater(t, result, 0)
		assert.Less(t, result, jobCount)
		assert.Less(t, errors, jobCount)
	})

	t.Run("Drain Jobs on Stop", func(t *testing.T) {
		wg := NewWorkerGroup[int](10, 100)

		jobCount := 100
		jobs := make([]uint, jobCount)
		for i := 0; i < jobCount; i++ {
			jobs[i] = 1
		}

		var result int
		var errors int
		ctx := context.Background()

		var waits sync.WaitGroup

		waits.Add(1)
		go func() {
			RunJobs(ctx, wg, jobs, jobFunc, resultFuncWrapper(&result, &errors))
			waits.Done()
		}()

		waits.Add(1)
		go func() {
			// calling Stop quickly after starting the jobs should result in
			// completion of some jobs, but not all
			// the job queue should also drain completely and terminate
			<-time.After(300 * time.Millisecond)
			wg.Stop()
			waits.Done()
		}()

		waits.Wait()

		assert.Greater(t, result, 0)
		assert.Less(t, result, jobCount)
	})

	t.Run("Stop w/ Small Queue", func(t *testing.T) {
		wg := NewWorkerGroup[int](10, 10)

		jobCount := 100
		jobs := make([]uint, jobCount)
		for i := 0; i < jobCount; i++ {
			jobs[i] = 1
		}

		var result int
		var errors int
		ctx := context.Background()

		var waits sync.WaitGroup

		waits.Add(1)
		go func() {
			RunJobs(ctx, wg, jobs, jobFunc, resultFuncWrapper(&result, &errors))
			waits.Done()
		}()

		waits.Add(1)
		go func() {
			// calling Stop quickly after starting the jobs should result in
			// completion of some jobs, but not all
			// the job queue should also drain completely and terminate
			<-time.After(300 * time.Millisecond)
			wg.Stop()
			waits.Done()
		}()

		waits.Wait()

		assert.Greater(t, errors, 0)
		assert.Greater(t, result, 0)
		assert.Less(t, result, jobCount)
		assert.Less(t, errors, jobCount)

	})
}

func BenchmarkWorkerGroup(b *testing.B) {
	procs := runtime.GOMAXPROCS(0)

	b.Run("MaxProcs", func(b *testing.B) {
		wg := NewWorkerGroup[bool](procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		var count int
		go func(w *WorkerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.NotifyResult(0):
					for _, r := range w.Results(0) {
						if r.Data {
							count++
						}
					}
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_ = wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(100 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			}, 0)
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})

	b.Run("2x_MaxProcs", func(b *testing.B) {
		wg := NewWorkerGroup[bool](2*procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		var count int
		go func(w *WorkerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.NotifyResult(0):
					for _, r := range w.Results(0) {
						if r.Data {
							count++
						}
					}
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_ = wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(100 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			}, 0)
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})

	b.Run("10x_MaxProcs", func(b *testing.B) {
		wg := NewWorkerGroup[bool](10*procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		var count int
		go func(w *WorkerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.NotifyResult(0):
					for _, r := range w.Results(0) {
						if r.Data {
							count++
						}
					}
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			_ = wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(100 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			}, 0)
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})
}
