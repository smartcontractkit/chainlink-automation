package keepers

import (
	"context"
	"fmt"
	"log"
	"runtime"
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
		cache:    newCache[types.UpkeepResult](20 * time.Minute), // TODO: default expiration should be configured based on block time
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

	var cacheHits int

	// - check upkeeps selected
	sample := []*types.UpkeepResult{}
	for i := 0; i < size; i++ {
		// skip if reported
		result, cached := s.cache.Get(string(keys[i]))
		if cached {
			cacheHits++
			if result.State == Perform {
				sample = append(sample, &result)
			}
		} else {
			// perform check and update cache with result
			s.logger.Printf("checking upkeep %s", keys[i])
			ok, u, _ := s.registry.CheckUpkeep(ctx, types.Address([]byte{}), keys[i])
			if ok {
				sample = append(sample, &u)
			}
			s.cache.Set(string(u.Key), u, defaultExpiration)
		}
	}

	s.logger.Printf("sampling cache hit ratio %d/%d", cacheHits, size)

	// - return array of results
	return sample, nil
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
