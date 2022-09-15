package keepers

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestWorkerGroup(t *testing.T) {
	wg := newWorkerGroup[bool](8, 1000)
	ctx, cancel := context.WithCancel(context.Background())

	go func(w *workerGroup[bool], c context.Context) {
		for {
			select {
			case <-w.results:
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

	<-time.After(20 * time.Millisecond)
	cancel()
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
