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

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type onDemandUpkeepService struct {
	logger           *log.Logger
	ratio            sampleRatio
	headSubscriber   types.HeadSubscriber
	registry         types.Registry
	shuffler         shuffler[types.UpkeepKey]
	cache            *cache[types.UpkeepResult]
	cacheCleaner     *intervalCacheCleaner[types.UpkeepResult]
	samplingResults  samplingUpkeepsResults
	samplingDuration time.Duration
	workers          *workerGroup[types.UpkeepResults]
	stopProcs        chan struct{}
}

// newOnDemandUpkeepService provides an object that implements the UpkeepService
// by running a worker pool that makes RPC network calls every time upkeeps
// need to be sampled. This variant has limitations in how quickly large numbers
// of upkeeps can be checked. Be aware that network calls are not rate limited
// from this service.
func newOnDemandUpkeepService(
	ratio sampleRatio,
	headSubscriber types.HeadSubscriber,
	registry types.Registry,
	logger *log.Logger,
	samplingDuration time.Duration,
	cacheExpire time.Duration,
	cacheClean time.Duration,
	workers int,
	workerQueueLength int,
) *onDemandUpkeepService {
	s := &onDemandUpkeepService{
		logger:           logger,
		ratio:            ratio,
		headSubscriber:   headSubscriber,
		registry:         registry,
		samplingDuration: samplingDuration,
		shuffler:         new(cryptoShuffler[types.UpkeepKey]),
		cache:            newCache[types.UpkeepResult](cacheExpire),
		workers:          newWorkerGroup[types.UpkeepResults](workers, workerQueueLength),
		stopProcs:        make(chan struct{}),
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

func (s *onDemandUpkeepService) SampleUpkeeps(_ context.Context, filters ...func(types.UpkeepKey) bool) (types.UpkeepResults, error) {
	if s.workers == nil {
		panic("cannot sample upkeeps without runner")
	}

	results := s.samplingResults.get()
	if len(results) == 0 {
		return nil, nil
	}

	filteredResults := make(types.UpkeepResults, 0, len(results))

EachKey:
	for _, result := range results {
		for _, filter := range filters {
			if !filter(result.Key) {
				s.logger.Printf("filtered out key '%s'", result.Key)
				continue EachKey
			}
		}

		filteredResults = append(filteredResults, result)
	}

	return filteredResults, nil
}

func (s *onDemandUpkeepService) CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) (types.UpkeepResults, error) {
	var (
		wg                sync.WaitGroup
		results           = make([]types.UpkeepResult, len(keys))
		nonCachedKeysIdxs []int
		nonCachedKeys     []types.UpkeepKey
	)

	for i, key := range keys {
		wg.Add(1)
		go func(i int, key types.UpkeepKey) {
			// the cache is a collection of keys (block & id) that map to cached
			// results. if the same upkeep is checked at a block that has already been
			// checked, return the cached result
			if result, cached := s.cache.Get(string(key)); cached {
				results[i] = result
			} else {
				nonCachedKeysIdxs = append(nonCachedKeysIdxs, i)
				nonCachedKeys = append(nonCachedKeys, key)
			}
			wg.Done()
		}(i, key)
	}

	wg.Wait()

	// check upkeep at block number in key
	// return result including performData
	checkResults, err := s.registry.CheckUpkeep(ctx, nonCachedKeys...)
	if err != nil {
		return nil, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	// Cache results
	for i, u := range checkResults {
		s.cache.Set(string(keys[nonCachedKeysIdxs[i]]), u, defaultExpiration)
		results[nonCachedKeysIdxs[i]] = u
	}

	return results, nil
}

func (s *onDemandUpkeepService) start() {
	// TODO: if this process panics, restart it
	go s.cacheCleaner.Run(s.cache)
	go func() {
		if err := s.runSamplingUpkeeps(); err != nil {
			s.logger.Fatal(err)
		}
	}()
}

func (s *onDemandUpkeepService) stop() {
	close(s.stopProcs)
	close(s.cacheCleaner.stop)
}

func (s *onDemandUpkeepService) runSamplingUpkeeps() error {
	ctx, cancel := context.WithCancel(context.Background())

	headTriggerCh := make(chan struct{}, 1)
	defer close(headTriggerCh)

	// Start the sampling upkeep process for heads
	go func() {
		for range headTriggerCh {
			s.processLatestHead(ctx)
		}
	}()

	// Cancel context when receiving the stop signal
	go func() {
		<-s.stopProcs
		cancel()
	}()

	return s.headSubscriber.OnNewHead(ctx, func(_ types.BlockKey) {
		// This is needed in order to do not block the process when a new head comes in.
		// The running upkeep sampling process should be finished first before starting
		// sampling for the next head.
		select {
		case headTriggerCh <- struct{}{}:
		default:
		}
	})
}

// processLatestHead performs checking upkeep logic for all eligible keys of the given head
func (s *onDemandUpkeepService) processLatestHead(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, s.samplingDuration)
	defer cancel()

	// Get only the active upkeeps from the contract. This should not include
	// any cancelled upkeeps.
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, "0")
	if err != nil {
		s.samplingResults.purge()
		s.logger.Printf("%s: failed to get upkeeps from registry for sampling", err)
		return
	}

	s.logger.Printf("%d active upkeep keys found in registry", len(keys))
	if len(keys) == 0 {
		s.samplingResults.purge()
		return
	}

	// select x upkeeps at random from set
	keys = s.shuffler.Shuffle(keys)
	size := s.ratio.OfInt(len(keys))

	s.logger.Printf("%d results selected by provided ratio %s", size, s.ratio)
	if size <= 0 {
		s.samplingResults.purge()
		return
	}

	upkeepResults, err := s.parallelCheck(ctx, keys[:size])
	if err != nil {
		s.samplingResults.purge()
		s.logger.Printf("%s: failed to parallel check upkeeps", err)
		return
	}

	s.samplingResults.set(upkeepResults)
}

func (s *onDemandUpkeepService) parallelCheck(ctx context.Context, keys []types.UpkeepKey) (types.UpkeepResults, error) {
	samples := newSyncedArray[types.UpkeepResult]()

	if len(keys) == 0 {
		return samples.Values(), nil
	}

	var (
		cacheHits      int
		wg             sync.WaitGroup
		wResults       workerResults
		keysToSendLock sync.Mutex
		keysToSend     = make([]types.UpkeepKey, 0, len(keys))
	)

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
		wg.Add(1)
		go func(key types.UpkeepKey) {
			defer wg.Done()

			// no RPC lookups need to be done if a result has already been cached
			result, cached := s.cache.Get(string(key))
			if cached {
				cacheHits++
				if result.State == types.Eligible {
					samples.Append(result)
				}
				return
			}

			// Add key to the slice that is going to be sent to the worker queue
			keysToSendLock.Lock()
			keysToSend = append(keysToSend, key)
			keysToSendLock.Unlock()
		}(key)
	}
	wg.Wait()

	// Create batches from the given keys.
	// Max 10 items in the batch.
	keysBatches := createBatches(keysToSend, 20)
	for _, batch := range keysBatches {
		// for every job added to the worker queue, add to the wait group
		// all jobs should complete before completing the function
		s.logger.Printf("attempting to send keys to worker group")
		wg.Add(1)
		if err := s.workers.Do(ctx, makeWorkerFunc(ctx, s.logger, s.registry, batch)); err != nil {
			if !errors.Is(err, ErrContextCancelled) {
				// the worker process has probably stopped so the function
				// should terminate with an error
				close(done)
				return nil, fmt.Errorf("%w: failed to add upkeep check to worker queue", err)
			}

			// if the context is cancelled before the work can be
			// finished, stop adding work and allow existing to finish.
			// cancelling this context will cancel all waiting worker
			// functions and have them report immediately. this will
			// result in a lot of errors in the results collector.
			s.logger.Printf("context cancelled while attempting to add to queue")
			wg.Done()
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
	if wResults.Total() > 0 && wResults.Total() == wResults.Failures() && wResults.LastErr() != nil {
		return samples.Values(), fmt.Errorf("%w: last error encounter by worker was '%s'", ErrTooManyErrors, wResults.LastErr())
	}

	return samples.Values(), nil
}

func (s *onDemandUpkeepService) aggregateWorkerResults(w *sync.WaitGroup, r *workerResults, sa *syncedArray[types.UpkeepResult], done chan struct{}) {
	s.logger.Printf("starting service to read worker results")

Outer:
	for {
		select {
		case result := <-s.workers.results:
			if result.Err == nil {
				r.AddSuccess(1)

				// Cache results
				for i := range result.Data {
					res := result.Data[i]
					s.cache.Set(string(res.Key), res, defaultExpiration)
					if res.State == types.Eligible {
						sa.Append(res)
					}
				}
			} else {
				r.SetLastErr(result.Err)
				s.logger.Printf("error received from worker result: %s", result.Err)
				r.AddFailure(1)
			}
			w.Done()
		case <-done:
			s.logger.Printf("done signal received for job result aggregator")
			break Outer
		}
	}
}

func makeWorkerFunc(jobCtx context.Context, logger *log.Logger, registry types.Registry, keys []types.UpkeepKey) work[types.UpkeepResults] {
	keysStr := upkeepKeysToString(keys)
	logger.Printf("check upkeep job created for keys: %s", keysStr)
	return func(serviceCtx context.Context) (types.UpkeepResults, error) {
		// cancel the job if either the worker group is stopped (ctx) or the
		// job context is cancelled (jobCtx)
		if jobCtx.Err() != nil || serviceCtx.Err() != nil {
			return nil, fmt.Errorf("job not attempted because one of two contexts had already been cancelled")
		}

		c, cancel := context.WithCancel(jobCtx)
		done := make(chan struct{}, 1)

		go func() {
			select {
			case <-jobCtx.Done():
				logger.Printf("check upkeep job context cancelled for keys: %s", keysStr)
				cancel()
			case <-serviceCtx.Done():
				logger.Printf("worker service context cancelled")
				cancel()
			case <-done:
				cancel()
			}
		}()

		start := time.Now()

		// perform check and update cache with result
		checkResults, err := registry.CheckUpkeep(c, keys...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep keys: %s", err, keysStr)
		} else {
			logger.Printf("check %d upkeeps took %dms to perform", len(keys), time.Since(start)/time.Millisecond)

			for _, result := range checkResults {
				if result.State == types.Eligible {
					logger.Printf("upkeep ready to perform for key %s", result.Key)
				}

				if result.FailureReason != 0 {
					logger.Printf("upkeep '%s' had a non-zero failure reason: %d", result.Key, result.FailureReason)
				}
			}
		}

		// close go-routine to prevent memory leaks
		done <- struct{}{}

		return checkResults, err
	}
}

type workerResults struct {
	success int
	failure int
	lastErr error
	mu      sync.RWMutex
}

func (wr *workerResults) AddSuccess(amt int) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.success = wr.success + amt
}

func (wr *workerResults) Failures() int {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.failure
}

func (wr *workerResults) LastErr() error {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.lastErr
}

func (wr *workerResults) AddFailure(amt int) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.failure = wr.failure + amt
}

func (wr *workerResults) SetLastErr(err error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	wr.lastErr = err
}

func (wr *workerResults) Total() int {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return wr.total()
}

func (wr *workerResults) total() int {
	return wr.success + wr.failure
}

func (wr *workerResults) SuccessRate() float64 {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return float64(wr.success) / float64(wr.total())
}

func (wr *workerResults) FailureRate() float64 {
	wr.mu.RLock()
	defer wr.mu.RUnlock()
	return float64(wr.failure) / float64(wr.total())
}

type samplingUpkeepsResults struct {
	upkeepResults types.UpkeepResults
	sync.Mutex
}

func (sur *samplingUpkeepsResults) purge() {
	sur.Lock()
	sur.upkeepResults = make(types.UpkeepResults, 0)
	sur.Unlock()
}

func (sur *samplingUpkeepsResults) set(results types.UpkeepResults) {
	sur.Lock()
	sur.upkeepResults = make(types.UpkeepResults, len(results))
	copy(sur.upkeepResults, results)
	sur.Unlock()
}

func (sur *samplingUpkeepsResults) get() types.UpkeepResults {
	sur.Lock()
	results := make(types.UpkeepResults, len(sur.upkeepResults))
	copy(results, sur.upkeepResults)
	sur.upkeepResults = make(types.UpkeepResults, 0)
	sur.Unlock()

	return results
}
