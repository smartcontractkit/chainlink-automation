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
	// - get all upkeeps from contract
	s.logger.Printf("get all active upkeep keys")
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get upkeeps from registry for sampling", err)
	}

	s.logger.Printf("%d upkeep keys found in registry", len(keys))
	// - select x upkeeps at random from set
	keys = s.shuffler.Shuffle(keys)
	size := s.ratio.OfInt(len(keys))
	s.logger.Printf("%d keys selected by provided ratio", size)

	// - check upkeeps selected
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

	var cacheHits int
	var wg sync.WaitGroup

	// start the results listener
	// if the context provided is cancelled, this listener shouldn't terminate
	// until all results have been collected. each worker is also passed this
	// function's context and will terminate when that context is cancelled
	// resulting in multiple errors being collected by this listener.
	done := make(chan struct{})
	go func() {
		s.logger.Printf("starting service to read worker results")
		var success int
		var failure int

	Outer:
		for {
			select {
			case result := <-s.workers.results:
				wg.Done()
				if result.Err == nil {
					success++
					// cache results
					s.cache.Set(string(result.Data.Key), result.Data, defaultExpiration)
					if result.Data.State == types.Eligible {
						samples.Append(&result.Data)
					}
				} else {
					failure++
				}
			case <-done:
				s.logger.Printf("done signal received")
				break Outer
			}
		}

		s.logger.Printf("worker call success rate: %f; failure rate: %f", float64(success)/float64(success+failure), float64(failure)/float64(success+failure))
	}()

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

	s.logger.Printf("sampling cache hit ratio %d/%d", cacheHits, len(keys))

	return samples.Values(), nil
}

func makeWorkerFunc(logger *log.Logger, registry types.Registry, key types.UpkeepKey, jobCtx context.Context) func(ctx context.Context) (types.UpkeepResult, error) {
	return func(ctx context.Context) (types.UpkeepResult, error) {
		// cancel the job if either the worker group is stopped (ctx) or the
		// job context is cancelled (jobCtx)
		if jobCtx.Err() != nil || ctx.Err() != nil {
			return types.UpkeepResult{}, fmt.Errorf("job not completed because one of two contexts had already been cancelled")
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

		if err != nil {
			logger.Printf("error checking upkeep '%s': %s", key, err)
		}

		if u.FailureReason != 0 {
			logger.Printf("upkeep '%s' had a non-zero failure reason: %d", key, u.FailureReason)
		}

		// close go-routine to prevent memory leaks
		done <- struct{}{}

		return u, err
	}
}
