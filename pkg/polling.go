package ocr2keepers

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/encoder"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type KeyStatusCoordinator interface {
	IsPending(types.UpkeepKey) (bool, error)
}

type KeyProvider interface {
	ActiveKeys(context.Context) ([]types.UpkeepKey, error)
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
	filterer keepers.Coordinator,
	eligibilityProvider encoder.EligibilityProvider,
) *PollingObserver {
	ctx, cancel := context.WithCancel(context.Background())

	ob := &PollingObserver{
		ctx:                 ctx,
		cancel:              cancel,
		logger:              logger,
		workers:             pkgutil.NewWorkerGroup[types.UpkeepResults](workers, workerQueueLength),
		workerBatchLimit:    10, // TODO: hard coded for now
		samplingDuration:    maxSamplingDuration,
		registry:            registry,
		keys:                keys,
		shuffler:            util.Shuffler[types.UpkeepKey]{Source: util.NewCryptoRandSource()}, // use crypto/rand shuffling for true random
		ratio:               ratio,
		stager:              &stager{},
		coord:               coord,
		cache:               pkgutil.NewCache[types.UpkeepResult](cacheExpire),
		cacheCleaner:        pkgutil.NewIntervalCacheCleaner[types.UpkeepResult](cacheClean),
		filterer:            filterer,
		eligibilityProvider: eligibilityProvider,
	}

	// make all go-routines started by this entity automatically recoverable
	// on panics
	ob.services = []Service{
		util.NewRecoverableService(&simpleService{f: ob.runHeadTasks, c: cancel}, logger),
		// TODO: workers is not recoverable because it cannot restart yet
		util.NewRecoverableService(&simpleService{f: func() error { return nil }, c: func() { ob.workers.Stop() }}, logger),
	}

	// automatically stop all services if the reference is no longer reachable
	// this is a safety in the case Stop isn't called explicitly
	runtime.SetFinalizer(ob, func(srv *PollingObserver) { srv.Stop() })

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
	coord               KeyStatusCoordinator        // key status coordinator tracks in-flight status
	shuffler            Shuffler[types.UpkeepKey]   // provides shuffling logic for upkeep keys
	filterer            keepers.Coordinator         // provides filtering logic for upkeep keys
	eligibilityProvider encoder.EligibilityProvider // provides an eligibility check for upkeep keys

}

// Observe implements the Observer interface and provides a slice of identifiers
// that were observed to be performable along with the block at which they were
// observed. All ids that are pending are filtered out.
func (bso *PollingObserver) Observe() (types.BlockKey, []types.UpkeepIdentifier, error) {
	bl, ids := bso.stager.get()
	filteredIDs := make([]types.UpkeepIdentifier, 0, len(ids))

	for _, id := range ids {
		key := chain.NewUpkeepKeyFromBlockAndID(bl, id)

		if pending, err := bso.coord.IsPending(key); pending || err != nil {
			bso.logger.Printf("error checking pending state for '%s': %s", key, err)
			continue
		}

		if !bso.filterer.IsPending(key) {
			bso.logger.Printf("filtered out key '%s'", key)
			continue
		}

		filteredIDs = append(filteredIDs, id)
	}

	return bl, filteredIDs, nil
}

// CheckUpkeep implements the Observer interface. It takes an number of upkeep
// keys and returns upkeep results.
func (bso *PollingObserver) CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error) {
	var (
		results           = make([]types.UpkeepResult, len(keys))
		nonCachedKeysIdxs = make([]int, 0, len(keys))
		nonCachedKeys     = make([]types.UpkeepKey, 0, len(keys))
	)

	for i, key := range keys {
		// the cache is a collection of keys (block & id) that map to cached
		// results. if the same upkeep is checked at a block that has already been
		// checked, return the cached result
		if result, cached := bso.cache.Get(key.String()); cached {
			results[i] = result
		} else {
			nonCachedKeysIdxs = append(nonCachedKeysIdxs, i)
			nonCachedKeys = append(nonCachedKeys, key)
		}
	}

	// All keys are cached
	if len(nonCachedKeys) == 0 {
		return results, nil
	}

	// check upkeep at block number in key
	// return result including performData
	checkResults, err := bso.registry.CheckUpkeep(ctx, nonCachedKeys...)
	if err != nil {
		return nil, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	// Cache results
	for i, u := range checkResults {
		bso.cache.Set(keys[nonCachedKeysIdxs[i]].String(), u, pkgutil.DefaultCacheExpiration)
		results[nonCachedKeysIdxs[i]] = u
	}

	return results, nil
}

// Start will start all required internal services. Calling this function again
// after the first is a noop.
func (bso *PollingObserver) Start() {
	bso.startOnce.Do(func() {
		for _, svc := range bso.services {
			svc.Start()
		}
	})
}

// Stop will stop all internal services allowing the observer to exit cleanly.
func (bso *PollingObserver) Stop() {
	bso.stopOnce.Do(func() {
		for _, svc := range bso.services {
			svc.Stop()
		}
	})
}

func (bso *PollingObserver) runHeadTasks() error {
	ch := bso.heads.HeadTicker()
	for {
		select {
		case bl := <-ch:
			// limit the context timeout to configured value
			ctx, cancel := context.WithTimeout(bso.ctx, bso.samplingDuration)

			// run sampling with latest head
			bso.processLatestHead(ctx, bl)

			// clean up resources by canceling the context after processing
			cancel()
		case <-bso.ctx.Done():
			return bso.ctx.Err()
		}
	}
}

// processLatestHead performs checking upkeep logic for all eligible keys of the given head
func (bso *PollingObserver) processLatestHead(ctx context.Context, blockKey types.BlockKey) {
	var (
		keys []types.UpkeepKey
		err  error
	)

	// Get only the active upkeeps from the key provider. This should not include
	// any cancelled upkeeps.
	if keys, err = bso.keys.ActiveKeys(ctx); err != nil {
		bso.logger.Printf("%s: failed to get upkeeps from registry for sampling", err)
		return
	}

	bso.logger.Printf("%d active upkeep keys found in registry", len(keys))

	// reduce keys to ratio size and shuffle. this can return a nil array.
	// in that case we have no keys so return.
	if keys = bso.shuffleAndSliceKeysToRatio(keys); keys == nil {
		return
	}

	bso.stager.prepareBlock(blockKey)

	// run checkupkeep on all keys. an error from this function should
	// bubble up.
	if err = bso.parallelCheck(ctx, keys); err != nil {
		bso.logger.Printf("%s: failed to parallel check upkeeps", err)
		return
	}

	// advance the staged block/upkeep id list to the next in line
	bso.stager.advance()
}

func (bso *PollingObserver) shuffleAndSliceKeysToRatio(keys []types.UpkeepKey) []types.UpkeepKey {
	keys = bso.shuffler.Shuffle(keys)
	size := bso.ratio.OfInt(len(keys))

	if len(keys) == 0 || size <= 0 {
		return nil
	}

	bso.logger.Printf("%d results selected by provided ratio %s", size, bso.ratio)

	return keys[:size]
}

func (bso *PollingObserver) parallelCheck(ctx context.Context, keys []types.UpkeepKey) error {
	if len(keys) == 0 {
		return nil
	}

	var wResults util.Results

	// Create batches from the given keys.
	// Max keyBatchSize items in the batch.
	pkgutil.RunJobs(
		ctx,
		bso.workers,
		util.Unflatten(keys, bso.workerBatchLimit),
		bso.wrapWorkerFunc(),
		bso.wrapAggregate(&wResults),
	)

	if wResults.Total() == 0 {
		bso.logger.Printf("no network calls were made for this sampling set")
	} else {
		bso.logger.Printf("worker call success rate: %.2f; failure rate: %.2f; total calls %d", wResults.SuccessRate(), wResults.FailureRate(), wResults.Total())
	}

	// multiple network calls can result in an error while some can be successful
	// in the case that all workers encounter an error, bubble this up as a hard
	// failure of the process.
	if wResults.Total() > 0 && wResults.Total() == wResults.Failures && wResults.Err != nil {
		return fmt.Errorf("%w: last error encounter by worker was '%s'", ErrTooManyErrors, wResults.Err)
	}

	return nil
}

func (bso *PollingObserver) wrapWorkerFunc() func(context.Context, []types.UpkeepKey) (types.UpkeepResults, error) {
	return func(ctx context.Context, keys []types.UpkeepKey) (types.UpkeepResults, error) {
		start := time.Now()

		// perform check and update cache with result
		checkResults, err := bso.registry.CheckUpkeep(ctx, keys...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep keys: %s", err, keys)
		} else {
			bso.logger.Printf("check %d upkeeps took %dms to perform", len(keys), time.Since(start)/time.Millisecond)

			for _, result := range checkResults {
				if bso.eligibilityProvider.Eligible(result) {
					bso.logger.Printf("upkeep ready to perform for key %s", result.Key)
				} else {
					bso.logger.Printf("upkeep '%s' is not eligible with failure reason: %d", result.Key, result.FailureReason)
				}
			}
		}

		return checkResults, err
	}
}

func (bso *PollingObserver) wrapAggregate(r *util.Results) func(types.UpkeepResults, error) {
	return func(result types.UpkeepResults, err error) {
		if err == nil {
			r.Successes++

			for i := range result {
				res := result[i]

				bso.cache.Set(string(res.Key.String()), res, pkgutil.DefaultCacheExpiration)

				if bso.eligibilityProvider.Eligible(result[i]) {
					_, id, err := result[i].Key.BlockKeyAndUpkeepID()
					if err != nil {
						continue
					}

					bso.stager.prepareIdentifier(id)
				}
			}
		} else {
			r.Err = err
			bso.logger.Printf("error received from worker result: %s", err)
			r.Failures++
		}
	}
}

type stager struct {
	currentIDs   []types.UpkeepIdentifier
	currentBlock types.BlockKey
	nextIDs      []types.UpkeepIdentifier
	nextBlock    types.BlockKey
	sync.RWMutex
}

func (st *stager) prepareBlock(bl types.BlockKey) {
	st.Lock()
	defer st.Unlock()

	st.nextBlock = bl
}

func (st *stager) prepareIdentifier(id types.UpkeepIdentifier) {
	st.Lock()
	defer st.Unlock()

	if st.nextIDs == nil {
		st.nextIDs = []types.UpkeepIdentifier{}
	}

	st.nextIDs = append(st.nextIDs, id)
}

func (st *stager) advance() {
	st.Lock()
	defer st.Unlock()

	st.currentBlock = st.nextBlock
	st.currentIDs = make([]types.UpkeepIdentifier, len(st.nextIDs))

	copy(st.currentIDs, st.nextIDs)

	st.nextIDs = make([]types.UpkeepIdentifier, 0)
}

func (st *stager) get() (types.BlockKey, []types.UpkeepIdentifier) {
	st.RLock()
	defer st.RUnlock()

	return st.currentBlock, st.currentIDs
}
