package executer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
)

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type Registry interface {
	CheckUpkeep(context.Context, bool, ...ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error)
}

type Encoder interface {
	// Eligible determines if an upkeep is eligible or not. This allows an
	// upkeep result to be abstract and only the encoder is able and responsible
	// for decoding it.
	Eligible(ocr2keepers.UpkeepResult) (bool, error)
	// Detail is a temporary value that provides upkeep key and gas to perform.
	// A better approach might be needed here.
	Detail(ocr2keepers.UpkeepResult) (ocr2keepers.UpkeepKey, uint32, error)
	// SplitUpkeepKey ...
	SplitUpkeepKey(ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error)
}

type Executer struct {
	// injected dependencies
	logger *log.Logger
	reg    Registry
	enc    Encoder

	// initialized by the constructor
	workers      *pkgutil.WorkerGroup[[]ocr2keepers.UpkeepResult] // parallelizer for RPC calls
	cache        *pkgutil.Cache[ocr2keepers.UpkeepResult]
	cacheCleaner *pkgutil.IntervalCacheCleaner[ocr2keepers.UpkeepResult]

	// configurations
	workerBatchLimit int // the maximum number of items in RPC batch call

	// run state data
	mu       sync.Mutex
	runState int
}

func NewExecuter(
	logger *log.Logger,
	reg Registry,
	enc Encoder,
	workers int, // maximum number of workers in worker group
	workerQueueLength int, // size of worker queue; set to approximately the number of items expected in workload
	cacheExpire time.Duration,
	cacheClean time.Duration,
) (*Executer, error) {
	return &Executer{
		logger:           logger,
		reg:              reg,
		enc:              enc,
		workers:          pkgutil.NewWorkerGroup[[]ocr2keepers.UpkeepResult](workers, workerQueueLength),
		cache:            pkgutil.NewCache[ocr2keepers.UpkeepResult](cacheExpire),
		cacheCleaner:     pkgutil.NewIntervalCacheCleaner[ocr2keepers.UpkeepResult](cacheClean),
		workerBatchLimit: 10,
	}, nil
}

func (o *Executer) CheckUpkeep(ctx context.Context, mercuryEnabled bool, keys ...ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error) {
	r, err := o.parallelCheck(ctx, mercuryEnabled, keys)
	if err != nil {
		return nil, err
	}

	return r.Values(), nil
}

func (o *Executer) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.runState == 0 {
		go o.cacheCleaner.Run(o.cache)
		o.runState = 1
	}

	return nil
}

func (o *Executer) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.runState == 1 {
		o.cacheCleaner.Stop()
		o.workers.Stop()
	}

	return nil
}

// parallelCheck should be satisfied by the executer
func (o *Executer) parallelCheck(ctx context.Context, mercuryEnabled bool, keys []ocr2keepers.UpkeepKey) (*Result, error) {
	result := NewResult()

	if len(keys) == 0 {
		return result, nil
	}

	toRun := make([]ocr2keepers.UpkeepKey, 0, len(keys))
	for _, key := range keys {

		// if in cache, add to result
		if res, ok := o.cache.Get(string(key)); ok {
			result.Add(res)
			continue
		}

		// else add to lookup job
		toRun = append(toRun, key)
	}

	// no more to do
	if len(toRun) == 0 {
		return result, nil
	}

	// Create batches from the given keys.
	// Max keyBatchSize items in the batch.
	pkgutil.RunJobs(
		ctx,
		o.workers,
		util.Unflatten(toRun, o.workerBatchLimit),
		o.wrapWorkerFunc(mercuryEnabled),
		o.wrapAggregate(result),
	)

	if result.Total() == 0 {
		o.logger.Printf("no network calls were made for this sampling set")
	} else {
		o.logger.Printf("worker call success rate: %.2f; failure rate: %.2f; total calls %d", result.SuccessRate(), result.FailureRate(), result.Total())
	}

	// multiple network calls can result in an error while some can be successful
	// in the case that all workers encounter an error, bubble this up as a hard
	// failure of the process.
	if result.Total() > 0 && result.Total() == result.Failures() && result.Err() != nil {
		return nil, fmt.Errorf("%w: last error encounter by worker was '%s'", ErrTooManyErrors, result.Err())
	}

	return result, nil
}

func (o *Executer) wrapWorkerFunc(mercuryEnabled bool) func(context.Context, []ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error) {
	return func(ctx context.Context, keys []ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error) {
		start := time.Now()

		// perform check and update cache with result
		checkResults, err := o.reg.CheckUpkeep(ctx, mercuryEnabled, keys...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep keys: %s", err, keys)
		} else {
			o.logger.Printf("check %d upkeeps took %dms to perform", len(keys), time.Since(start)/time.Millisecond)

			for _, result := range checkResults {
				ok, err := o.enc.Eligible(result)
				if err != nil {
					o.logger.Printf("eligibility check error: %s", err)
					continue
				}

				// TODO: ok might be assumed here???
				if ok {
					// TODO: try something other than using `Detail`
					key, _, _ := o.enc.Detail(result)
					o.logger.Printf("upkeep ready to perform for key %s", key)
				}
			}
		}

		return checkResults, err
	}
}

func (o *Executer) wrapAggregate(r *Result) func([]ocr2keepers.UpkeepResult, error) {
	return func(result []ocr2keepers.UpkeepResult, err error) {
		if err == nil {
			r.AddSuccesses(1)

			for _, res := range result {
				// TODO: find another way to do this
				key, _, _ := o.enc.Detail(res)
				// TODO: using string again
				o.cache.Set(string(key), res, pkgutil.DefaultCacheExpiration)

				r.Add(res)
			}
		} else {
			r.SetErr(err)
			o.logger.Printf("error received from worker result: %s", err)
			r.AddFailures(1)
		}
	}
}
