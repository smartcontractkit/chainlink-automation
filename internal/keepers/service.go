package keepers

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type simpleUpkeepService struct {
	logger       *log.Logger
	ratio        sampleRatio
	registry     types.Registry
	shuffler     shuffler[types.UpkeepKey]
	cache        *cache[types.UpkeepResult]
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
		logger:   logger,
		ratio:    ratio,
		registry: registry,
		shuffler: new(cryptoShuffler[types.UpkeepKey]),
		cache:    newCache[types.UpkeepResult](cacheExpire),
		workers:  newWorkerGroup[types.UpkeepResult](workers, workerQueueLength),
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

	result, cached := s.cache.Get(string(key))
	if cached {
		return result, nil
	}

	// check upkeep at block number in key
	// return result including performData
	ok, u, err := s.registry.CheckUpkeep(ctx, key)
	if err != nil {
		return types.UpkeepResult{}, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	result = types.UpkeepResult{
		Key:   key,
		State: types.Skip,
	}

	if ok {
		result.State = types.Perform
		result.PerformData = u.PerformData
	}

	s.cache.Set(string(key), result, defaultExpiration)

	return result, nil
}

func (s *simpleUpkeepService) SetUpkeepState(ctx context.Context, uk types.UpkeepKey, state types.UpkeepState) error {
	var err error

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

	// start the channel listener
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
					if result.Data.State == types.Perform {
						sample = append(sample, &result.Data)
					}
				} else {
					failure++
				}
			case <-done:
				break Outer
			case <-ctx.Done():
				break Outer
			}
		}

		s.logger.Printf("worker call success rate: %f; failure rate: %f", float64(success)/float64(success+failure), float64(failure)/float64(success+failure))
	}()

	// go through keys and check the cache first
	// if an item doesn't exist on the cache, send the items to the worker threads
	for _, key := range keys {
		// skip if reported
		result, cached := s.cache.Get(string(key))
		if cached {
			cacheHits++
			if result.State == types.Perform {
				sample = append(sample, &result)
			}
		} else {
			wg.Add(1)
			if err := s.workers.Do(ctx, makeWorkerFunc(s.logger, s.registry, key)); err != nil {
				return nil, err
			}
		}
	}

	wg.Wait()
	close(done)

	s.logger.Printf("sampling cache hit ratio %d/%d", cacheHits, len(keys))

	return sample, nil
}

func makeWorkerFunc(logger *log.Logger, registry types.Registry, key types.UpkeepKey) func(ctx context.Context) (types.UpkeepResult, error) {
	return func(ctx context.Context) (types.UpkeepResult, error) {
		// perform check and update cache with result
		logger.Printf("checking upkeep %s", key)
		ok, u, err := registry.CheckUpkeep(ctx, key)
		if ok {
			logger.Printf("upkeep ready to perform for key %s", key)
		}

		if err != nil {
			logger.Printf("error checking upkeep '%s': %s", key, err)
		}
		return u, err
	}
}
