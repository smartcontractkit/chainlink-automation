package execution

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/ocr2keepers/encoder"
	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type executer struct {
	logger *log.Logger

	registry types.Registry

	eligibilityProvider encoder.EligibilityProvider

	workers      *pkgutil.WorkerGroup[types.UpkeepResults]
	cache        *pkgutil.Cache[types.UpkeepResult]
	cacheCleaner *pkgutil.IntervalCacheCleaner[types.UpkeepResult]

	rpcBatchLimit int
}

var _ types.Executer = &executer{}

func NewExecuter(
	logger *log.Logger,
	registry types.Registry,
	eligibilityProvider encoder.EligibilityProvider,
	cacheExpire time.Duration,
	cacheClean time.Duration,
	workers int, // maximum number of workers in worker group
	workerQueueSize int, // size of worker queue; set to approximately the number of items expected in workload
	rpcBatchLimit int,
) *executer {
	return &executer{
		logger: logger,

		registry: registry,

		eligibilityProvider: eligibilityProvider,

		workers:      pkgutil.NewWorkerGroup[types.UpkeepResults](workers, workerQueueSize),
		cache:        pkgutil.NewCache[types.UpkeepResult](cacheExpire),
		cacheCleaner: pkgutil.NewIntervalCacheCleaner[types.UpkeepResult](cacheClean),

		rpcBatchLimit: rpcBatchLimit,
	}
}

// Run executes checkUpkeep for the given keys and their checkData that is used in log upkeeps scenario
func (ex *executer) Run(ctx context.Context, keys []types.UpkeepKey, checkData [][]byte) (types.UpkeepResults, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	var wResults util.Results

	pkgutil.RunJobs(
		ctx,
		ex.workers,
		util.Unflatten(keys, ex.rpcBatchLimit),
		ex.wrapWorkerFunc(),
		ex.wrapAggregate(&wResults),
	)

	if wResults.Total() == 0 {
		ex.logger.Printf("no network calls were made for this sampling set")
	} else {
		ex.logger.Printf("worker call success rate: %.2f; failure rate: %.2f; total calls %d", wResults.SuccessRate(), wResults.FailureRate(), wResults.Total())
	}

	// multiple network calls can result in an error while some can be successful
	// in the case that all workers encounter an error, bubble this up as a hard
	// failure of the process.
	if wResults.Total() > 0 && wResults.Total() == wResults.Failures && wResults.Err != nil {
		return nil, fmt.Errorf("%w: last error encounter by worker was '%s'", ErrTooManyErrors, wResults.Err)
	}

	results := make([]types.UpkeepResult, 0)
	for _, key := range keys {
		res, ok := ex.cache.Get(key.String())
		if !ok {
			// TODO: handle an upkeep that was not populated in cache?
			continue
		}
		results = append(results, res)
	}

	return results, nil
}

func (ex *executer) wrapWorkerFunc() func(context.Context, []types.UpkeepKey) (types.UpkeepResults, error) {
	return func(ctx context.Context, keys []types.UpkeepKey) (types.UpkeepResults, error) {
		start := time.Now()

		// perform check and update cache with result
		// TODO: check data?
		checkResults, err := ex.checkUpkeeps(ctx, keys...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep keys: %s", err, keys)
		} else {
			ex.logger.Printf("check %d upkeeps took %dms to perform", len(keys), time.Since(start)/time.Millisecond)

			for _, result := range checkResults {
				if ex.eligibilityProvider.Eligible(result) {
					ex.logger.Printf("upkeep ready to perform for key %s", result.Key)
				} else {
					ex.logger.Printf("upkeep '%s' is not eligible with failure reason: %d", result.Key, result.FailureReason)
				}
			}
		}

		return checkResults, err
	}
}

func (ex *executer) wrapAggregate(r *util.Results) func(types.UpkeepResults, error) {
	return func(result types.UpkeepResults, err error) {
		if err == nil {
			r.Successes++
		} else {
			r.Err = err
			ex.logger.Printf("error received from worker result: %s", err)
			r.Failures++
		}
	}
}

// checkUpkeeps does an upkeep check for the given keys, it utilizes a cache to avoid redundant calls.
func (ex *executer) checkUpkeeps(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error) {
	var (
		results           = make([]types.UpkeepResult, len(keys))
		nonCachedKeysIdxs = make([]int, 0, len(keys))
		nonCachedKeys     = make([]types.UpkeepKey, 0, len(keys))
	)

	for i, key := range keys {
		if result, cached := ex.cache.Get(key.String()); cached {
			results[i] = result
		} else {
			nonCachedKeysIdxs = append(nonCachedKeysIdxs, i)
			nonCachedKeys = append(nonCachedKeys, key)
		}
	}

	// no execution required, all keys are cached
	if len(nonCachedKeys) == 0 {
		return results, nil
	}

	// check upkeep at block number in key
	// return result including performData
	checkResults, err := ex.registry.CheckUpkeep(ctx, false, nonCachedKeys...)
	if err != nil {
		return nil, fmt.Errorf("%w: service failed to check upkeep from registry", err)
	}

	for i, u := range checkResults {
		ex.cache.Set(keys[nonCachedKeysIdxs[i]].String(), u, pkgutil.DefaultCacheExpiration)
		results[nonCachedKeysIdxs[i]] = u
	}

	return results, nil
}
