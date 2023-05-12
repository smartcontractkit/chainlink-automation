package keepers

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/observer"
	"github.com/smartcontractkit/ocr2keepers/pkg/ratio"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var (
	ErrTooManyErrors          = fmt.Errorf("too many errors in parallel worker process")
	ErrSamplingNotInitialized = fmt.Errorf("sampling not initialized")
)

type onDemandUpkeepService struct {
	logger           *log.Logger
	sampleRatio      ratio.SampleRatio
	headSubscriber   types.HeadSubscriber
	registry         types.Registry
	shuffler         shuffler[types.UpkeepIdentifier]
	cache            *util.Cache[types.UpkeepResult]
	cacheCleaner     *util.IntervalCacheCleaner[types.UpkeepResult]
	samplingResults  samplingUpkeepsResults
	samplingDuration time.Duration
	workers          *util.WorkerGroup[types.UpkeepResults]
	observers        []observer.Observer
	ctx              context.Context
	cancel           context.CancelFunc
	mercuryEnabled   bool
}

// newOnDemandUpkeepService provides an object that implements the UpkeepService
// by running a worker pool that makes RPC network calls every time upkeeps
// need to be sampled. This variant has limitations in how quickly large numbers
// of upkeeps can be checked. Be aware that network calls are not rate limited
// from this service.
func newOnDemandUpkeepService(
	sampleRatio ratio.SampleRatio,
	headSubscriber types.HeadSubscriber,
	registry types.Registry,
	logger *log.Logger,
	samplingDuration time.Duration,
	cacheExpire time.Duration,
	cacheClean time.Duration,
	workers int,
	workerQueueLength int,
	mercuryEnabled bool,
	observers []observer.Observer,
) *onDemandUpkeepService {
	ctx, cancel := context.WithCancel(context.Background())
	s := &onDemandUpkeepService{
		logger:           logger,
		sampleRatio:      sampleRatio,
		headSubscriber:   headSubscriber,
		registry:         registry,
		samplingDuration: samplingDuration,
		shuffler:         new(cryptoShuffler[types.UpkeepIdentifier]),
		cache:            util.NewCache[types.UpkeepResult](cacheExpire),
		cacheCleaner:     util.NewIntervalCacheCleaner[types.UpkeepResult](cacheClean),
		workers:          util.NewWorkerGroup[types.UpkeepResults](workers, workerQueueLength),
		observers:        observers,
		ctx:              ctx,
		cancel:           cancel,
		mercuryEnabled:   mercuryEnabled,
	}

	// stop the cleaner go-routine once the upkeep service is no longer reachable
	runtime.SetFinalizer(s, func(srv *onDemandUpkeepService) { srv.stop() })

	// start background services
	s.start()

	return s
}

var _ upkeepService = (*onDemandUpkeepService)(nil)

func (s *onDemandUpkeepService) SampleUpkeeps(_ context.Context, filters ...func(types.UpkeepKey) bool) (types.BlockKey, types.UpkeepResults, error) {
	s.logger.Printf("onDemandUpkeepService.SampleUpkeeps called")

	blockKey, results, ok := s.samplingResults.get()
	if !ok {
		s.logger.Printf("sampling not initialized")
		return nil, nil, ErrSamplingNotInitialized
	}

	s.logger.Printf("onDemandUpkeepService.SampleUpkeeps sampled blockKey %s and %d results", blockKey.String(), len(results))

	filteredResults := make(types.UpkeepResults, 0, len(results))

EachKey:
	for _, result := range results {
		for _, filter := range filters {
			if !filter(result.Key) {
				s.logger.Printf("filtered out key during SampleUpkeeps '%s'", result.Key)
				continue EachKey
			}
		}

		filteredResults = append(filteredResults, result)
	}

	s.logger.Printf("onDemandUpkeepService.SampleUpkeeps returning blockKey %s and %d results", blockKey.String(), len(filteredResults))

	return blockKey, filteredResults, nil
}

func (s *onDemandUpkeepService) CheckUpkeep(ctx context.Context, mercuryEnabled bool, keys ...types.UpkeepKey) (types.UpkeepResults, error) {
	var (
		wg                sync.WaitGroup
		results           = make([]types.UpkeepResult, len(keys))
		nonCachedKeysLock sync.Mutex
		nonCachedKeysIdxs = make([]int, 0, len(keys))
		nonCachedKeys     = make([]types.UpkeepKey, 0, len(keys))
	)

	for i, key := range keys {
		wg.Add(1)
		go func(i int, key types.UpkeepKey) {
			// the cache is a collection of keys (block & id) that map to cached
			// results. if the same upkeep is checked at a block that has already been
			// checked, return the cached result
			if result, cached := s.cache.Get(key.String()); cached {
				results[i] = result
			} else {
				nonCachedKeysLock.Lock()
				nonCachedKeysIdxs = append(nonCachedKeysIdxs, i)
				nonCachedKeys = append(nonCachedKeys, key)
				nonCachedKeysLock.Unlock()
			}
			wg.Done()
		}(i, key)
	}

	wg.Wait()

	// All keys are cached
	if len(nonCachedKeys) == 0 {
		return results, nil
	}

	// check upkeep at block number in key
	// return result including performData
	checkResults, err := s.registry.CheckUpkeep(ctx, mercuryEnabled, nonCachedKeys...)
	if err != nil {
		return nil, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	// Cache results
	for i, u := range checkResults {
		s.cache.Set(keys[nonCachedKeysIdxs[i]].String(), u, util.DefaultCacheExpiration)
		results[nonCachedKeysIdxs[i]] = u
	}

	return results, nil
}

func (s *onDemandUpkeepService) start() {
	// TODO: if this process panics, restart it
	go s.cacheCleaner.Run(s.cache)
}

func (s *onDemandUpkeepService) stop() {
	s.cancel()
	s.workers.Stop()
	s.cacheCleaner.Stop()
}

type samplingUpkeepsResults struct {
	upkeepResults types.UpkeepResults
	blockKey      types.BlockKey
	ok            bool
	sync.Mutex
}

func (sur *samplingUpkeepsResults) set(blockKey types.BlockKey, results types.UpkeepResults) {
	sur.Lock()
	defer sur.Unlock()

	sur.upkeepResults = make(types.UpkeepResults, len(results))
	copy(sur.upkeepResults, results)
	sur.blockKey = blockKey
	sur.ok = true
}

func (sur *samplingUpkeepsResults) get() (types.BlockKey, types.UpkeepResults, bool) {
	sur.Lock()
	defer sur.Unlock()

	return sur.blockKey, sur.upkeepResults, sur.ok
}
