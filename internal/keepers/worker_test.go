package keepers

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	t.Run("Do-Not Add to Queue on Context Cancel", func(t *testing.T) {
		w := &worker[int]{
			Name:  "worker",
			Queue: make(chan *worker[int]), // leave this unbuffered to not allow anything to block the queue
		}

		// channel for work results to be added
		results := make(chan workResult[int], 1)

		f := func(_ context.Context) (int, error) {
			return 10, fmt.Errorf("error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go w.Do(ctx, results, f)

		// wait a short period to ensure enough time for result to be returned
		<-time.After(20 * time.Millisecond)
		cancel()

		tmr := time.NewTimer(100 * time.Millisecond)
		select {
		case r := <-results:
			tmr.Stop()
			// should have a result
			assert.Equal(t, 10, r.Data, "data from work result should match")
			assert.Equal(t, fmt.Errorf("error"), r.Err, "error from work result should match")
		case <-tmr.C:
			// fail
			assert.Fail(t, "work result not placed on result channel")
		}
	})

	t.Run("Do-Not Add to Results on Context Cancel", func(t *testing.T) {
		w := &worker[int]{
			Name:  "worker",
			Queue: make(chan *worker[int]), // leave this unbuffered to not allow anything to block the queue
		}

		// channel for work results to be added; unbuffered to block
		results := make(chan workResult[int])

		f := func(_ context.Context) (int, error) {
			return 10, fmt.Errorf("error")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go w.Do(ctx, results, f)

		// wait a short period to ensure enough time for result to be returned
		<-time.After(20 * time.Millisecond)
		cancel()

		tmr := time.NewTimer(100 * time.Millisecond)
		select {
		case <-results:
			assert.Fail(t, "work result was placed on result channel")
			tmr.Stop()
		case <-tmr.C:
			// fail
			break
		}
	})
}

func TestWorkerGroup(t *testing.T) {

	t.Run("All Work Done", func(t *testing.T) {
		wg := newWorkerGroup[bool](8, 1000)
		ctx := context.Background()

		closed := make(chan struct{})
		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				tmr := time.NewTimer(50 * time.Millisecond)
				select {
				case <-w.results:
					tmr.Stop()
					done++
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
			})
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
		wg := newWorkerGroup[bool](8, 1000)
		ctx := context.Background()

		<-time.After(1500 * time.Millisecond)

		closed := make(chan struct{})
		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				tmr := time.NewTimer(50 * time.Millisecond)
				select {
				case <-w.results:
					tmr.Stop()
					done++
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
			})
			if err != nil {
				t.FailNow()
			}
		}

		<-closed

		assert.Equal(t, 10, done)
	})

	t.Run("Error on Cancel and Full Queue", func(t *testing.T) {
		wg := &workerGroup[int]{
			queue: make(chan work[int]), // unbuffered to block
			stop:  make(chan struct{}),
		}
		ctx, cancel := context.WithCancel(context.Background())

		errors := make(chan error, 1)
		go func() {
			err := wg.Do(ctx, func(_ context.Context) (int, error) {
				return 1, nil
			})
			errors <- err
		}()

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
		wg := &workerGroup[int]{
			queue: make(chan work[int]), // unbuffered to block
			stop:  make(chan struct{}),
		}

		errors := make(chan error, 1)
		go func() {
			err := wg.Do(context.Background(), func(_ context.Context) (int, error) {
				return 1, nil
			})
			errors <- err
		}()

		// wait for a short period to ensure function is in select statement
		<-time.After(20 * time.Millisecond)
		close(wg.stop)

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
		wg := &workerGroup[int]{
			queue: make(chan work[int]), // unbuffered to block
			stop:  make(chan struct{}),
		}
		ctx, cancel := context.WithCancel(context.Background())

		errors := make(chan error, 1)
		cancel()
		go func() {
			err := wg.Do(ctx, func(_ context.Context) (int, error) {
				return 1, nil
			})
			errors <- err
		}()

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
		wg := newWorkerGroup[int](1, 1)

		go func() {
			err := wg.Do(context.Background(), func(ctx context.Context) (int, error) {
				tmr := time.NewTimer(100 * time.Millisecond)
				select {
				case <-ctx.Done():
					tmr.Stop()
					return 0, fmt.Errorf("error")
				case <-tmr.C:
					return 1, nil
				}
			})

			assert.NoError(t, err, "adding work to group should not return error")
		}()

		// give some time for queue to start work
		<-time.After(20 * time.Millisecond)

		// stop process to close the queue and cancel the run context
		wg.Stop()

		// check results to see that work item returned short due to context cancel
		// may not have results, which is expected behavior
		tmr := time.NewTimer(150 * time.Millisecond)
		select {
		case r := <-wg.results:
			tmr.Stop()
			// should have a result
			assert.Equal(t, 0, r.Data, "data from work result should match")
			assert.Equal(t, fmt.Errorf("error"), r.Err, "error from work result should match")
		case <-tmr.C:
			break
		}

		// check that queue is closed
		testAdd := func() {
			wg.queue <- func(_ context.Context) (int, error) { return 0, nil }
		}
		assert.Panics(t, testAdd, "queue should be closed")
	})
}

func BenchmarkWorkerGroup(b *testing.B) {
	procs := runtime.GOMAXPROCS(0)

	b.Run("MaxProcs", func(b *testing.B) {
		wg := newWorkerGroup[bool](procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		var count int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case r := <-w.results:
					if r.Data {
						count++
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
			})
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})

	b.Run("2x_MaxProcs", func(b *testing.B) {
		wg := newWorkerGroup[bool](2*procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		var count int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case r := <-w.results:
					if r.Data {
						count++
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
			})
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})

	b.Run("10x_MaxProcs", func(b *testing.B) {
		wg := newWorkerGroup[bool](10*procs, 10)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		var count int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case r := <-w.results:
					if r.Data {
						count++
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
			})
		}
		b.StopTimer()

		<-time.After(10 * time.Millisecond)
		cancel()
		b.ReportMetric(float64(count), "complete")
		b.ReportMetric(0, "ns/op")
	})
}
