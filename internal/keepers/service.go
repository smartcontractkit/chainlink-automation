package keepers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type onDemandUpkeepService struct {
	logger       *log.Logger
	ratio        sampleRatio
	registry     types.Registry
	shuffler     shuffler[types.UpkeepKey]
	cache        *cache[types.UpkeepResult]
	cacheCleaner *intervalCacheCleaner[types.UpkeepResult]
	workers      *workerGroup[types.UpkeepResult]
	stopProcs    chan struct{}
}

// newOnDemandUpkeepService provides an object that implements the UpkeepService
// by running a worker pool that makes RPC network calls every time upkeeps
// need to be sampled. This variant has limitations in how quickly large numbers
// of upkeeps can be checked. Be aware that network calls are not rate limited
// from this service.
func newOnDemandUpkeepService(ratio sampleRatio, registry types.Registry, logger *log.Logger, cacheExpire time.Duration, cacheClean time.Duration, workers int, workerQueueLength int) *onDemandUpkeepService {
	s := &onDemandUpkeepService{
		logger:    logger,
		ratio:     ratio,
		registry:  registry,
		shuffler:  new(cryptoShuffler[types.UpkeepKey]),
		cache:     newCache[types.UpkeepResult](cacheExpire),
		workers:   newWorkerGroup[types.UpkeepResult](workers, workerQueueLength),
		stopProcs: make(chan struct{}),
	}

	cl := &intervalCacheCleaner[types.UpkeepResult]{
		Interval: cacheClean,
		stop:     make(chan struct{}),
	}

	s.cacheCleaner = cl

	// stop the cleaner go-routine once the upkeep service is no longer reachable
	runtime.SetFinalizer(s, func(srv *onDemandUpkeepService) { srv.stop() })

	// start background services
	s.start()
	return s
}

var _ upkeepService = (*onDemandUpkeepService)(nil)

func (s *onDemandUpkeepService) SampleUpkeeps(ctx context.Context, filters ...func(types.UpkeepKey) bool) ([]*types.UpkeepResult, error) {
	// get only the active upkeeps from the contract. this should not include
	// any cancelled upkeeps
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get upkeeps from registry for sampling", err)
	}

	s.logger.Printf("%d active upkeep keys found in registry", len(keys))
	if len(keys) == 0 {
		return []*types.UpkeepResult{}, nil
	}

	// select x upkeeps at random from set
	keys = s.shuffler.Shuffle(keys)
	size := s.ratio.OfInt(len(keys))

	s.logger.Printf("%d keys selected by provided ratio %.8f", size, s.ratio)
	if size <= 0 {
		return []*types.UpkeepResult{}, nil
	}

	if s.workers == nil {
		panic("cannot sample upkeeps without runner")
	}

	return s.parallelCheck(ctx, keys[:size], filters...)
}

func (s *onDemandUpkeepService) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (types.UpkeepResult, error) {
	var result types.UpkeepResult

	// the cache is a collection of keys (block & id) that map to cached
	// results. if the same upkeep is checked at a block that has already been
	// checked, return the cached result
	result, cached := s.cache.Get(string(key))
	if cached {
		return result, nil
	}

	// check upkeep at block number in key
	// return result including performData
	_, u, err := s.registry.CheckUpkeep(ctx, key)
	if err != nil {
		return types.UpkeepResult{}, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	s.cache.Set(string(key), u, defaultExpiration)

	return u, nil
}

func (s *onDemandUpkeepService) start() {
	// TODO: if this process panics, restart it
	go s.cacheCleaner.Run(s.cache)
}

func (s *onDemandUpkeepService) stop() {
	close(s.stopProcs)
	close(s.cacheCleaner.stop)
}

func (s *onDemandUpkeepService) parallelCheck(ctx context.Context, keys []types.UpkeepKey, filters ...func(types.UpkeepKey) bool) ([]*types.UpkeepResult, error) {
	samples := newSyncedArray[*types.UpkeepResult]()

	if len(keys) == 0 {
		return samples.Values(), nil
	}

	var cacheHits int
	var wg sync.WaitGroup
	var wResults workerResults

	// start the results listener
	// if the context provided is cancelled, this listener shouldn't terminate
	// until all results have been collected. each worker is also passed this
	// function's context and will terminate when that context is cancelled
	// resulting in multiple errors being collected by this listener.
	done := make(chan struct{})
	go s.aggregateWorkerResults(&wg, &wResults, samples, done)

	// go through keys and check the cache first
	// if an item doesn't exist on the cache, send the items to the worker threads
	for _, key := range keys {
		add := true
		for _, filter := range filters {
			if !filter(key) {
				add = false
				break
			}
		}

		if !add {
			continue
		}

		// no RPC lookups need to be done if a result has already been cached
		result, cached := s.cache.Get(string(key))
		if cached {
			cacheHits++
			if result.State == types.Eligible {
				samples.Append(&result)
			}
			continue
		}

		// for every job added to the worker queue, add to the wait group
		// all jobs should complete before completing the function
		wg.Add(1)
		if err := s.workers.Do(ctx, makeWorkerFunc(s.logger, s.registry, key, ctx)); err != nil {
			if errors.Is(err, ErrContextCancelled) {
				// if the context is cancelled before the work can be
				// finished, stop adding work and allow existing to finish.
				// cancelling this context will cancel all waiting worker
				// functions and have them report immediately. this will
				// result in a lot of errors in the results collector.
				s.logger.Printf("context cancelled while attempting to add to queue")
				wg.Done()
				break
			}

			// the worker process has probably stopped so the function
			// should terminate with an error
			close(done)
			return nil, fmt.Errorf("%w: failed to add upkeep check to worker queue", err)
		}
	}

	s.logger.Printf("waiting for results to be read")
	wg.Wait()
	close(done)

	if wResults.Total() == 0 {
		s.logger.Printf("no network calls were made for this sampling set")
	} else {
		s.logger.Printf("worker call success rate: %.2f; failure rate: %.2f; total calls %d", wResults.SuccessRate(), wResults.FailureRate(), wResults.Total())
	}

	s.logger.Printf("sampling cache hit ratio %d/%d", cacheHits, len(keys))

	// multiple network calls can result in an error while some can be successful
	// in the case that all workers encounter an error, bubble this up as a hard
	// failure of the process.
	if wResults.Total() > 0 && wResults.Total() == wResults.Failure && wResults.LastErr != nil {
		return samples.Values(), fmt.Errorf("%w: too many errors in parallel worker process; last error provided", wResults.LastErr)
	}

	return samples.Values(), nil
}

func (s *onDemandUpkeepService) aggregateWorkerResults(w *sync.WaitGroup, r *workerResults, sa *syncedArray[*types.UpkeepResult], done chan struct{}) {
	s.logger.Printf("starting service to read worker results")

Outer:
	for {
		select {
		case result := <-s.workers.results:
			w.Done()
			if result.Err == nil {
				r.Success++
				// cache results
				s.cache.Set(string(result.Data.Key), result.Data, defaultExpiration)
				if result.Data.State == types.Eligible {
					sa.Append(&result.Data)
				}
			} else {
				r.LastErr = result.Err
				s.logger.Printf("error received from worker result: %s", result.Err)
				r.Failure++
			}
		case <-done:
			s.logger.Printf("done signal received for worker group")
			break Outer
		}
	}
}

func makeWorkerFunc(logger *log.Logger, registry types.Registry, key types.UpkeepKey, jobCtx context.Context) func(ctx context.Context) (types.UpkeepResult, error) {
	return func(ctx context.Context) (types.UpkeepResult, error) {
		// cancel the job if either the worker group is stopped (ctx) or the
		// job context is cancelled (jobCtx)
		if jobCtx.Err() != nil || ctx.Err() != nil {
			return types.UpkeepResult{}, fmt.Errorf("job not attempted because one of two contexts had already been cancelled")
		}

		c, cancel := context.WithCancel(context.Background())
		done := make(chan struct{}, 1)

		go func() {
			select {
			case <-jobCtx.Done():
				logger.Printf("check upkeep job context cancelled for key %s", key)
				cancel()
			case <-ctx.Done():
				logger.Printf("worker service context cancelled")
				cancel()
			case <-done:
				cancel()
			}
		}()

		// perform check and update cache with result
		ok, u, err := registry.CheckUpkeep(c, key)
		if ok {
			logger.Printf("upkeep ready to perform for key %s", key)
		}

		if u.FailureReason != 0 {
			logger.Printf("upkeep '%s' had a non-zero failure reason: %d", key, u.FailureReason)
		}

		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep key '%s'", err, key)
		}

		// close go-routine to prevent memory leaks
		done <- struct{}{}

		return u, err
	}
}

type workerResults struct {
	Success int
	Failure int
	LastErr error
}

func (wr *workerResults) Total() int {
	return wr.Success + wr.Failure
}

func (wr *workerResults) SuccessRate() float64 {
	return float64(wr.Success) / float64(wr.Total())
}

func (wr *workerResults) FailureRate() float64 {
	return float64(wr.Failure) / float64(wr.Total())
}
