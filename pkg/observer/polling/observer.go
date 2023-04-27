package polling

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/encoder"
	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/observer"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type KeyStatusCoordinator interface {
	IsPending(types.UpkeepKey) (bool, error)
}

type KeyProvider interface {
	ActiveKeys(context.Context, types.BlockKey) ([]types.UpkeepKey, error)
}

type keyProvider struct {
	registry types.Registry
}

func NewKeyProvider(registry types.Registry) *keyProvider {
	return &keyProvider{
		registry: registry,
	}
}

func (p *keyProvider) ActiveKeys(ctx context.Context, blockKey types.BlockKey) ([]types.UpkeepKey, error) {
	upkeepIDs, err := p.registry.GetActiveUpkeepIDs(ctx)
	if err != nil {
		return nil, err
	}

	var upkeepKeys []types.UpkeepKey
	for _, k := range upkeepIDs {
		upkeepKeys = append(upkeepKeys, chain.NewUpkeepKeyFromBlockAndID(blockKey, k))
	}

	return upkeepKeys, nil
}

type Service interface {
	Start()
	Stop()
}

type Shuffler[T any] interface {
	Shuffle([]T) []T
}

// Ratio is an interface that provides functions to calculate a ratio of a given
// input
type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
	fmt.Stringer
}

func NewPollingObserver(
	logger *log.Logger,
	registry types.Registry,
	keys KeyProvider,
	ratio Ratio,
	workers int, // maximum number of workers in worker group
	workerQueueLength int, // size of worker queue; set to approximately the number of items expected in workload
	maxSamplingDuration time.Duration, // maximum amount of time allowed for RPC calls per head
	coord KeyStatusCoordinator,
	cacheExpire time.Duration,
	cacheClean time.Duration,
	filterer coordinator.Coordinator,
	eligibilityProvider encoder.EligibilityProvider,
	upkeepProvider encoder.UpkeepProvider,
	headSubscriber types.HeadSubscriber,
) *PollingObserver {
	ctx, cancel := context.WithCancel(context.Background())

	logger.Printf("NewPollingObserver instantiated")

	ob := &PollingObserver{
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		workers:          pkgutil.NewWorkerGroup[types.UpkeepResults](workers, workerQueueLength),
		workerBatchLimit: 10, // TODO: hard coded for now
		samplingDuration: maxSamplingDuration,
		registry:         registry,
		keys:             keys,
		heads:            headSubscriber,
		shuffler:         util.Shuffler[types.UpkeepKey]{Source: util.NewCryptoRandSource()}, // use crypto/rand shuffling for true random
		ratio:            ratio,
		stager: &stager{
			logger: logger,
		},
		coordinator:         coord,
		cache:               pkgutil.NewCache[types.UpkeepResult](cacheExpire),
		cacheCleaner:        pkgutil.NewIntervalCacheCleaner[types.UpkeepResult](cacheClean),
		filterer:            filterer,
		eligibilityProvider: eligibilityProvider,
		upkeepProvider:      upkeepProvider,
	}

	// make all go-routines started by this entity automatically recoverable
	// on panics
	ob.services = []Service{
		util.NewRecoverableService(&observer.SimpleService{F: ob.runHeadTasks, C: cancel}, logger),
		// TODO: workers is not recoverable because it cannot restart yet
		util.NewRecoverableService(&observer.SimpleService{F: func() error { return nil }, C: func() { ob.workers.Stop() }}, logger),
	}

	// automatically stop all services if the reference is no longer reachable
	// this is a safety in the case Stop isn't called explicitly
	runtime.SetFinalizer(ob, func(srv observer.Observer) { srv.Stop() })

	ob.Start()

	return ob
}

type PollingObserver struct {
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	stopOnce  sync.Once

	// static values provided by constructor
	workerBatchLimit int           // the maximum number of items in RPC batch call
	samplingDuration time.Duration // limits time spent processing a single block
	ratio            Ratio         // ratio for limiting sample size

	// initialized components inside a constructor
	services     []Service
	stager       *stager
	workers      *pkgutil.WorkerGroup[types.UpkeepResults] // parallelizer for RPC calls
	cache        *pkgutil.Cache[types.UpkeepResult]
	cacheCleaner *pkgutil.IntervalCacheCleaner[types.UpkeepResult]

	// dependency interfaces required by the polling observer
	logger              *log.Logger
	heads               types.HeadSubscriber        // provides new blocks to be operated on
	registry            types.Registry              // abstracted access to contract and chain
	keys                KeyProvider                 // provides keys to this block observer
	coordinator         KeyStatusCoordinator        // key status coordinator tracks in-flight status
	shuffler            Shuffler[types.UpkeepKey]   // provides shuffling logic for upkeep keys
	filterer            coordinator.Coordinator     // provides filtering logic for upkeep keys
	eligibilityProvider encoder.EligibilityProvider // provides an eligibility check for upkeep keys
	upkeepProvider      encoder.UpkeepProvider
}

func (o *PollingObserver) SetLogger(logger *log.Logger) {
	o.logger = logger
	o.stager.logger = logger
}

// Observe implements the Observer interface and provides a slice of identifiers
// that were observed to be performable along with the block at which they were
// observed. All ids that are pending are filtered out.
func (o *PollingObserver) Observe() (types.BlockKey, []types.UpkeepIdentifier, error) {
	o.logger.Printf("PollingObserver Observe called")

	bl, ids := o.stager.get()

	o.logger.Printf("PollingObserver Observe, got block %s and %d ids from stager", bl, len(ids))

	filteredIDs := make([]types.UpkeepIdentifier, 0, len(ids))

	for _, id := range ids {
		key := o.upkeepProvider.MakeUpkeepKey(bl, id)

		if pending, err := o.coordinator.IsPending(key); pending || err != nil {
			o.logger.Printf("PollingObserver Observe error checking pending state for '%s': %s", key, err)
			continue
		}

		if !o.filterer.IsPending(key) {
			o.logger.Printf("PollingObserver Observe filtered out key '%s'", key)
			continue
		}

		filteredIDs = append(filteredIDs, id)
	}

	o.logger.Printf("PollingObserver Observe, returning block %s and %d ids", bl, len(filteredIDs))

	return bl, filteredIDs, nil
}

// CheckUpkeep implements the Observer interface. It takes an number of upkeep
// keys and returns upkeep results.
func (o *PollingObserver) CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error) {
	o.logger.Printf("PollingObserver CheckUpkeep called for %d keys", len(keys))

	var (
		results           = make([]types.UpkeepResult, len(keys))
		nonCachedKeysIdxs = make([]int, 0, len(keys))
		nonCachedKeys     = make([]types.UpkeepKey, 0, len(keys))
	)

	for i, key := range keys {
		// the cache is a collection of keys (block & id) that map to cached
		// results. if the same upkeep is checked at a block that has already been
		// checked, return the cached result
		if result, cached := o.cache.Get(key.String()); cached {
			results[i] = result
		} else {
			nonCachedKeysIdxs = append(nonCachedKeysIdxs, i)
			nonCachedKeys = append(nonCachedKeys, key)
		}
	}

	// All keys are cached
	if len(nonCachedKeys) == 0 {
		o.logger.Printf("PollingObserver CheckUpkeep all keys are cached, returning")
		return results, nil
	}

	// check upkeep at block number in key
	// return result including performData
	checkResults, err := o.registry.CheckUpkeep(ctx, nonCachedKeys...)
	if err != nil {
		o.logger.Printf("PollingObserver CheckUpkeep errored: %s", err.Error())
		return nil, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	// Cache results
	for i, u := range checkResults {
		o.cache.Set(keys[nonCachedKeysIdxs[i]].String(), u, pkgutil.DefaultCacheExpiration)
		results[nonCachedKeysIdxs[i]] = u
	}

	o.logger.Printf("PollingObserver CheckUpkeep returning %d results", len(results))

	return results, nil
}

// Start will start all required internal services. Calling this function again
// after the first is a noop.
func (o *PollingObserver) Start() {
	o.logger.Printf("PollingObserver Start called")

	o.startOnce.Do(func() {
		for _, svc := range o.services {
			o.logger.Printf("PollingObserver service started")

			svc.Start()
		}
	})
}

// Stop will stop all internal services allowing the observer to exit cleanly.
func (o *PollingObserver) Stop() {
	o.stopOnce.Do(func() {
		for _, svc := range o.services {
			svc.Stop()
		}
	})
}

func (o *PollingObserver) runHeadTasks() error {
	ch := o.heads.HeadTicker()
	for {
		select {
		case bl := <-ch:
			o.logger.Printf("PollingObserver runHeadTasks block received")

			// limit the context timeout to configured value
			ctx, cancel := context.WithTimeout(o.ctx, o.samplingDuration)

			// run sampling with latest head
			o.processLatestHead(ctx, bl)

			// clean up resources by canceling the context after processing
			cancel()
		case <-o.ctx.Done():
			return o.ctx.Err()
		}
	}
}

// processLatestHead performs checking upkeep logic for all eligible keys of the given head
func (o *PollingObserver) processLatestHead(ctx context.Context, blockKey types.BlockKey) {
	var (
		keys []types.UpkeepKey
		err  error
	)

	o.logger.Printf("PollingObserver processLatestHead")

	// Get only the active upkeeps from the key provider. This should not include
	// any cancelled upkeeps.
	if keys, err = o.keys.ActiveKeys(ctx, blockKey); err != nil {
		o.logger.Printf("PollingObserver %s: failed to get upkeeps from registry for sampling", err)
		return
	}

	o.logger.Printf("PollingObserver %d active upkeep keys found in registry", len(keys))

	// reduce keys to ratio size and shuffle. this can return a nil array.
	// in that case we have no keys so return.
	if keys = o.shuffleAndSliceKeysToRatio(keys); keys == nil {
		o.logger.Printf("PollingObserver shuffle and slice returned nil keys, returning")
		return
	}

	o.logger.Printf("PollingObserver shuffled and sliced keys, %d keys remaining", len(keys))

	o.stager.prepareBlock(blockKey)

	o.logger.Printf("PollingObserver prepared block key %s", blockKey)

	// run checkupkeep on all keys. an error from this function should
	// bubble up.
	if err = o.parallelCheck(ctx, keys); err != nil {
		o.logger.Printf("PollingObserver %s: failed to parallel check upkeeps", err)
		return
	}

	// advance the staged block/upkeep id list to the next in line
	o.stager.advance()

	o.logger.Printf("PollingObserver advanced stager")
}

func (o *PollingObserver) shuffleAndSliceKeysToRatio(keys []types.UpkeepKey) []types.UpkeepKey {
	o.logger.Printf("about to shuffle and slice %d keys", len(keys))
	keys = o.shuffler.Shuffle(keys)
	o.logger.Printf("shuffle returned %d keys", len(keys))
	size := o.ratio.OfInt(len(keys))
	o.logger.Printf("ratio returned %d size", size)

	if len(keys) == 0 || size <= 0 {
		o.logger.Printf("have %d keys, size is %d, returning", len(keys), size)
		return nil
	}

	o.logger.Printf("%d results selected by provided ratio %s", size, o.ratio)

	return keys[:size]
}

func (o *PollingObserver) parallelCheck(ctx context.Context, keys []types.UpkeepKey) error {
	if len(keys) == 0 {
		return nil
	}

	var wResults util.Results

	// Create batches from the given keys.
	// Max keyBatchSize items in the batch.
	pkgutil.RunJobs(
		ctx,
		o.workers,
		util.Unflatten(keys, o.workerBatchLimit),
		o.wrapWorkerFunc(),
		o.wrapAggregate(&wResults),
	)

	if wResults.Total() == 0 {
		o.logger.Printf("no network calls were made for this sampling set")
	} else {
		o.logger.Printf("worker call success rate: %.2f; failure rate: %.2f; total calls %d", wResults.SuccessRate(), wResults.FailureRate(), wResults.Total())
	}

	// multiple network calls can result in an error while some can be successful
	// in the case that all workers encounter an error, bubble this up as a hard
	// failure of the process.
	if wResults.Total() > 0 && wResults.Total() == wResults.Failures && wResults.Err != nil {
		return fmt.Errorf("%w: last error encounter by worker was '%s'", ErrTooManyErrors, wResults.Err)
	}

	return nil
}

func (o *PollingObserver) wrapWorkerFunc() func(context.Context, []types.UpkeepKey) (types.UpkeepResults, error) {
	return func(ctx context.Context, keys []types.UpkeepKey) (types.UpkeepResults, error) {
		start := time.Now()

		// perform check and update cache with result
		checkResults, err := o.registry.CheckUpkeep(ctx, keys...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep keys: %s", err, keys)
		} else {
			o.logger.Printf("check %d upkeeps took %dms to perform", len(keys), time.Since(start)/time.Millisecond)

			for _, result := range checkResults {
				if o.eligibilityProvider.Eligible(result) {
					o.logger.Printf("upkeep ready to perform for key %s", result.Key)
				} else {
					o.logger.Printf("upkeep '%s' is not eligible with failure reason: %d", result.Key, result.FailureReason)
				}
			}
		}

		return checkResults, err
	}
}

func (o *PollingObserver) wrapAggregate(r *util.Results) func(types.UpkeepResults, error) {
	return func(result types.UpkeepResults, err error) {
		if err == nil {
			o.logger.Printf("PollingObserver wrapAggregate processing successful results")

			r.Successes++

			for _, res := range result {
				o.cache.Set(res.Key.String(), res, pkgutil.DefaultCacheExpiration)

				if o.eligibilityProvider.Eligible(res) {
					_, id, err := o.upkeepProvider.SplitUpkeepKey(res.Key)
					if err != nil {
						continue
					}

					o.stager.prepareIdentifier(id)
				} else {
					o.logger.Printf("PollingObserver wrapAggregate result not eligible")
				}
			}
		} else {
			r.Err = err
			o.logger.Printf("PollingObserver wrapAggregate error received from worker result: %s", err)
			r.Failures++
		}
	}
}

type stager struct {
	logger       *log.Logger
	currentIDs   []types.UpkeepIdentifier
	currentBlock types.BlockKey
	nextIDs      []types.UpkeepIdentifier
	nextBlock    types.BlockKey
	sync.RWMutex
}

func (s *stager) prepareBlock(block types.BlockKey) {
	s.Lock()
	defer s.Unlock()

	s.logger.Printf("stager prepareBlock called: %v", block)

	s.nextBlock = block
}

func (s *stager) prepareIdentifier(id types.UpkeepIdentifier) {
	s.Lock()
	defer s.Unlock()

	s.logger.Printf("stager prepareIdentifier called: %v", id)

	if s.nextIDs == nil {
		s.nextIDs = []types.UpkeepIdentifier{}
	}

	s.nextIDs = append(s.nextIDs, id)
}

func (s *stager) advance() {
	s.Lock()
	defer s.Unlock()

	s.logger.Printf("stager advance called")

	s.currentBlock = s.nextBlock
	s.currentIDs = make([]types.UpkeepIdentifier, len(s.nextIDs))

	copy(s.currentIDs, s.nextIDs)

	s.nextIDs = make([]types.UpkeepIdentifier, 0)
}

func (s *stager) get() (types.BlockKey, []types.UpkeepIdentifier) {
	s.RLock()
	defer s.RUnlock()

	s.logger.Printf("stager get called")

	return s.currentBlock, s.currentIDs
}
