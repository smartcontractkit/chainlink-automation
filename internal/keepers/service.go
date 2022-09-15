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
	ratio        SampleRatio
	registry     types.Registry
	shuffler     Shuffler[types.UpkeepKey]
	cache        *cache[types.UpkeepResult]
	cacheCleaner *intervalCacheCleaner[types.UpkeepResult]
	workers      *workerGroup[types.UpkeepResult]
}

// NewSimpleUpkeepService provides an object that implements the UpkeepService in a very
// rudamentary way. Sampling upkeeps is done on demand and completes in linear time with upkeeps.
//
// Cacheing is enabled such that subsequent checks/updates for the same key will not result in
// an RPC call.
//
// DO NOT USE THIS IN PRODUCTION
func NewSimpleUpkeepService(ratio SampleRatio, registry types.Registry, logger *log.Logger) *simpleUpkeepService {
	s := &simpleUpkeepService{
		logger:   logger,
		ratio:    ratio,
		registry: registry,
		shuffler: new(cryptoShuffler[types.UpkeepKey]),
		cache:    newCache[types.UpkeepResult](20 * time.Minute),                     // TODO: default expiration should be configured based on block time
		workers:  newWorkerGroup[types.UpkeepResult](10*runtime.GOMAXPROCS(0), 1000), // # of workers = 10 * [# of cpus]
	}

	cl := &intervalCacheCleaner[types.UpkeepResult]{
		Interval: 30 * time.Second, // TODO: update to sane default
		stop:     make(chan struct{}, 1),
	}

	s.cacheCleaner = cl
	go cl.Run(s.cache)

	// stop the cleaner go-routine once the upkeep service is no longer reachable
	runtime.SetFinalizer(s, func(srv *simpleUpkeepService) { srv.cacheCleaner.stop <- struct{}{} })

	return s
}

var _ UpkeepService = (*simpleUpkeepService)(nil)

func (s *simpleUpkeepService) SampleUpkeeps(ctx context.Context) ([]*types.UpkeepResult, error) {
	// - get all upkeeps from contract
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		// TODO: do better error bubbling
		return nil, err
	}

	// - select x upkeeps at random from set
	keys = s.shuffler.Shuffle(keys)
	size := s.ratio.OfInt(len(keys))

	// - check upkeeps selected
	if s.workers == nil {
		panic("cannot check upkeeps without runner")
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
	// TODO: which address should be passed to this function?
	ok, u, err := s.registry.CheckUpkeep(ctx, types.Address([]byte{}), key)
	if err != nil {
		// TODO: do better error bubbling
		return types.UpkeepResult{}, err
	}

	result = types.UpkeepResult{
		Key:   key,
		State: Skip,
	}

	if ok {
		result.State = Perform
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
		for {
			select {
			case result := <-s.workers.results:
				wg.Done()
				if result.Err == nil && result.Data.State == Perform {
					sample = append(sample, &result.Data)
				}
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// go through keys and check the cache first
	// if an item doesn't exist on the cache, send the items to the worker threads
	for _, key := range keys {
		// skip if reported
		result, cached := s.cache.Get(string(key))
		if cached {
			cacheHits++
			if result.State == Perform {
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
		_, u, err := registry.CheckUpkeep(ctx, types.Address([]byte{}), key)
		return u, err
	}
}
