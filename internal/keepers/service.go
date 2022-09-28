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

const defaultStaleWindow = 100 * 14 * time.Second // 100 blocks @ 14s per block

type simpleUpkeepService struct {
	logger       *log.Logger
	ratio        sampleRatio
	registry     types.Registry
	shuffler     shuffler[types.UpkeepKey]
	cache        *cache[types.UpkeepResult]
	stateCache   *cache[types.UpkeepState]
	cacheCleaner *intervalCacheCleaner[types.UpkeepResult]
	workers      *workerGroup[types.UpkeepResult]
}

// newSimpleUpkeepService provides an object that implements the UpkeepService in a very
// rudamentary way. Sampling upkeeps is done on demand and completes in linear time with upkeeps.
//
// Cacheing is enabled such that subsequent checks/updates for the same key will not result in
// an RPC call.
//
// DO NOT USE THIS IN PRODUCTION
func newSimpleUpkeepService(ratio sampleRatio, registry types.Registry, logger *log.Logger, cacheExpire time.Duration, cacheClean time.Duration, workers int, workerQueueLength int) *simpleUpkeepService {
	s := &simpleUpkeepService{
		logger:     logger,
		ratio:      ratio,
		registry:   registry,
		shuffler:   new(cryptoShuffler[types.UpkeepKey]),
		cache:      newCache[types.UpkeepResult](cacheExpire),
		stateCache: newCache[types.UpkeepState](defaultStaleWindow),
		workers:    newWorkerGroup[types.UpkeepResult](workers, workerQueueLength),
	}

	cl := &intervalCacheCleaner[types.UpkeepResult]{
		Interval: cacheClean,
		stop:     make(chan struct{}, 1),
	}

	s.cacheCleaner = cl
	go cl.Run(s.cache)

	// stop the cleaner go-routine once the upkeep service is no longer reachable
	runtime.SetFinalizer(s, func(srv *simpleUpkeepService) { srv.cacheCleaner.stop <- struct{}{} })

	return s
}

var _ upkeepService = (*simpleUpkeepService)(nil)

func (s *simpleUpkeepService) SampleUpkeeps(ctx context.Context) ([]*types.UpkeepResult, error) {
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
	return s.parallelCheck(ctx, keys[:size])
}

func (s *simpleUpkeepService) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (types.UpkeepResult, error) {
	var result types.UpkeepResult

	id, err := s.registry.IdentifierFromKey(key)
	if err != nil {
		return result, err
	}
	state, stateCached := s.stateCache.Get(string(id))

	result, cached := s.cache.Get(string(key))
	if cached {
		if stateCached {
			result.State = state
		}
		return result, nil
	}

	// check upkeep at block number in key
	// return result including performData
	needsPerform, u, err := s.registry.CheckUpkeep(ctx, key)
	if err != nil {
		return types.UpkeepResult{}, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	result = types.UpkeepResult{
		Key:   key,
		State: types.Skip,
	}

	if needsPerform {
		result.State = types.Perform
		result.PerformData = u.PerformData
	}

	if stateCached {
		result.State = state
	}

	s.cache.Set(string(key), result, defaultExpiration)

	return result, nil
}

func (s *simpleUpkeepService) SetUpkeepState(ctx context.Context, uk types.UpkeepKey, state types.UpkeepState) error {
	var err error

	id, err := s.registry.IdentifierFromKey(uk)
	if err != nil {
		return err
	}

	// set the state cache to the default expiration using the upkeep identifer
	// as the key
	s.stateCache.Set(string(id), state, defaultExpiration)

	result, cached := s.cache.Get(string(uk))
	if !cached {
		// if the value is not in the cache, do a hard check
		result, err = s.CheckUpkeep(ctx, uk)
		if err != nil {
			return fmt.Errorf("%w: cache miss and check for key '%s'", err, string(uk))
		}
	}

	result.State = state
	s.cache.Set(string(uk), result, defaultExpiration)
	return nil
}

func (s *simpleUpkeepService) parallelCheck(ctx context.Context, keys []types.UpkeepKey) ([]*types.UpkeepResult, error) {
	sample := make([]*types.UpkeepResult, 0, len(keys))
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
					if result.Data.State == types.Perform {
						sample = append(sample, &result.Data)
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
		// get reported status and continue if found
		id, err := s.registry.IdentifierFromKey(key)
		if err != nil {
			return sample, err
		}
		state, stateCached := s.stateCache.Get(string(id))
		if stateCached && state == types.Reported {
			cacheHits++
			continue
		}

		// skip if reported
		result, cached := s.cache.Get(string(key))
		if cached {
			cacheHits++
			if result.State == types.Perform {
				sample = append(sample, &result)
			}
			continue
		}

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

	return sample, nil
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
				cancel()
			case <-ctx.Done():
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

		// close go-routine to prevent memory leaks
		done <- struct{}{}

		return u, err
	}
}
