package keepers

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerGroup(t *testing.T) {

	t.Run("All Work Done", func(t *testing.T) {
		wg := newWorkerGroup[bool](8, 1000)
		ctx, cancel := context.WithCancel(context.Background())

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		for n := 0; n < 10; n++ {
			err := wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(10 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
			if err != nil {
				t.FailNow()
			}
		}

		<-time.After(30 * time.Millisecond)
		cancel()

		assert.Equal(t, 10, done)
	})

	t.Run("Wait Before Sending", func(t *testing.T) {
		// the worker group destroys workers when nothing is in the queue
		// for longer than 1 second. this test ensures that total number of
		// workers doesn't go below 0 causing the worker group to lock.
		wg := newWorkerGroup[bool](8, 1000)
		ctx, cancel := context.WithCancel(context.Background())

		<-time.After(1500 * time.Millisecond)

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		for n := 0; n < 10; n++ {
			err := wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(10 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
			if err != nil {
				t.FailNow()
			}
		}

		<-time.After(30 * time.Millisecond)
		cancel()

		assert.Equal(t, 10, done)
	})

	t.Run("Worker Shut Down", func(t *testing.T) {

		wg := newWorkerGroup[bool](8, 1000)
		ctx, cancel := context.WithCancel(context.Background())

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		for n := 0; n < 10; n++ {
			err := wg.Do(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(20 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
			if err != nil {
				t.FailNow()
			}
		}

		// wait for all workers to start
		<-time.After(10 * time.Millisecond)

		active := wg.activeWorkers
		assert.Greater(t, active, 0)

		// wait for at least 1 worker to be removed
		<-time.After(1200 * time.Millisecond)
		assert.Greater(t, active, wg.activeWorkers)
		cancel()

		assert.Equal(t, 10, done)
	})

	t.Run("Cancel Request On Full Queue", func(t *testing.T) {

		wg := newWorkerGroup[bool](1, 1)
		ctx, cancel := context.WithCancel(context.Background())

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		// add 2 items to the queue to set up the test
		// one will start executing and 1 will be waiting for a worker
		// to be available
		for n := 0; n < 3; n++ {
			// make the work queue wait for a bit
			err := wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
			if err != nil {
				t.FailNow()
			}
		}

		err := wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return true, nil
			case <-c.Done():
				return false, nil
			}
		})
		assert.ErrorIs(t, err, ErrQueueFull)

		cancel()
	})

	t.Run("No Add After Cancel", func(t *testing.T) {

		wg := newWorkerGroup[bool](1, 1)
		ctx, cancel := context.WithCancel(context.Background())

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		// first 2 will succeed
		for n := 0; n < 3; n++ {
			_ = wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
		}

		cancel()

		<-time.After(20 * time.Millisecond)

		// this should fail since the context was previously cancelled
		err := wg.Do(ctx, func(c context.Context) (bool, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return true, nil
			case <-c.Done():
				return false, nil
			}
		})
		assert.ErrorIs(t, err, ErrContextCancelled)

		// this should also fail
		err = wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return true, nil
			case <-c.Done():
				return false, nil
			}
		})
		assert.ErrorIs(t, err, ErrContextCancelled)
	})

	t.Run("No Add After Stop", func(t *testing.T) {

		wg := newWorkerGroup[bool](1, 1)
		ctx, cancel := context.WithCancel(context.Background())

		var done int
		go func(w *workerGroup[bool], c context.Context) {
			for {
				select {
				case <-w.results:
					done++
					continue
				case <-c.Done():
					return
				}
			}
		}(wg, ctx)

		// first 2 will succeed
		for n := 0; n < 3; n++ {
			_ = wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return true, nil
				case <-c.Done():
					return false, nil
				}
			})
		}

		<-time.After(20 * time.Millisecond)

		wg.Stop()

		// this should fail since the context was previously cancelled
		err := wg.Do(ctx, func(c context.Context) (bool, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return true, nil
			case <-c.Done():
				return false, nil
			}
		})
		assert.ErrorIs(t, err, ErrProcessStopped)

		// this should also fail
		err = wg.DoCancelOnFull(ctx, func(c context.Context) (bool, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return true, nil
			case <-c.Done():
				return false, nil
			}
		})
		assert.ErrorIs(t, err, ErrProcessStopped)

		cancel()
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
