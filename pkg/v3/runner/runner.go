package runner

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const WorkerBatchLimit int = 10

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

//go:generate mockery --name Runnable --structname MockRunnable --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/runner" --case underscore --filename runnable.generated.go
type Runnable interface {
	CheckUpkeeps(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

// ensure that the runner implements the same interface it consumes to indicate
// the runner simply wraps the underlying runnable with extra features
var _ Runnable = &Runner{}

// Runner is a component that parallelizes calls to the provided runnable both
// by batching tasks to individual calls as well as using parallel threads to
// execute calls to the runnable. All results are cached such that the same
// input job from a previous run will provide a cached response instead of
// calling the runnable.
//
// The Runner is structured as a direct replacement where the runnable is used
// as a dependency.
type Runner struct {
	// injected dependencies
	logger   *log.Logger
	runnable Runnable

	// initialized by the constructor
	workers      *pkgutil.WorkerGroup[[]ocr2keepers.CheckResult]        // parallelizer
	cache        *pkgutil.Cache[ocr2keepers.CheckResult]                // result cache
	cacheCleaner *pkgutil.IntervalCacheCleaner[ocr2keepers.CheckResult] // autmatic cache cleaner

	// configurations
	workerBatchLimit int // the maximum number of items in RPC batch call

	// run state data
	running atomic.Bool
	chClose chan struct{}
}

type RunnerConfig struct {
	// Workers is the maximum number of workers in worker group
	Workers int
	// WorkerQueueLength is size of worker queue; set to approximately the number of items expected in workload
	WorkerQueueLength int
	CacheExpire       time.Duration
	CacheClean        time.Duration
}

// NewRunner provides a new configured runner
func NewRunner(
	logger *log.Logger,
	runnable Runnable,
	conf RunnerConfig,
) (*Runner, error) {
	return &Runner{
		logger:           log.New(logger.Writer(), fmt.Sprintf("[%s | check-pipeline-runner]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		runnable:         runnable,
		workers:          pkgutil.NewWorkerGroup[[]ocr2keepers.CheckResult](conf.Workers, conf.WorkerQueueLength),
		cache:            pkgutil.NewCache[ocr2keepers.CheckResult](conf.CacheExpire),
		cacheCleaner:     pkgutil.NewIntervalCacheCleaner[ocr2keepers.CheckResult](conf.CacheClean),
		workerBatchLimit: WorkerBatchLimit,
		chClose:          make(chan struct{}, 1),
	}, nil
}

// CheckUpkeeps accepts an array of payloads, splits the workload into separate
// threads, executes the underlying runnable, and returns all results from all
// threads. If previous runs were already completed for the same one or more
// payloads, results will be pulled from the cache where available.
func (o *Runner) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	r, err := o.parallelCheck(ctx, payloads)
	if err != nil {
		return nil, err
	}

	return r.Values(), nil
}

// Start starts up the cache cleaner
func (o *Runner) Start(_ context.Context) error {
	if o.running.Load() {
		return fmt.Errorf("already running")
	}

	o.running.Swap(true)
	o.logger.Println("starting service")

	go o.cacheCleaner.Run(o.cache)

	<-o.chClose

	return nil
}

// Close stops the cache cleaner and the parallel worker process
func (o *Runner) Close() error {
	if !o.running.Load() {
		return fmt.Errorf("not running")
	}

	o.cacheCleaner.Stop()
	o.workers.Stop()
	o.running.Swap(false)

	o.chClose <- struct{}{}

	return nil
}

// parallelCheck should be satisfied by the Runner
func (o *Runner) parallelCheck(ctx context.Context, payloads []ocr2keepers.UpkeepPayload) (*result[ocr2keepers.CheckResult], error) {
	result := newResult[ocr2keepers.CheckResult]()

	if len(payloads) == 0 {
		return result, nil
	}

	toRun := make([]ocr2keepers.UpkeepPayload, 0, len(payloads))
	for _, payload := range payloads {

		// if in cache, add to result
		if res, ok := o.cache.Get(payload.WorkID); ok {
			result.Add(res)
			continue
		}

		// else add to lookup job
		toRun = append(toRun, payload)
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
		o.wrapWorkerFunc(),
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

func (o *Runner) wrapWorkerFunc() func(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	return func(ctx context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
		start := time.Now()

		allPayloadKeys := make([]string, len(payloads))
		for i := range payloads {
			allPayloadKeys[i] = payloads[i].WorkID
		}

		// perform check and update cache with result
		checkResults, err := o.runnable.CheckUpkeeps(ctx, payloads...)
		if err != nil {
			err = fmt.Errorf("%w: failed to check upkeep payloads for ids '%s'", err, strings.Join(allPayloadKeys, ", "))
		} else {
			o.logger.Printf("check %d upkeeps took %dms to perform", len(payloads), time.Since(start)/time.Millisecond)
		}

		return checkResults, err
	}
}

// UpkeepWorkID returns the identifier using the given upkeepID and trigger extension(tx hash and log index).
func UpkeepWorkID(id *big.Int, trigger ocr2keepers.Trigger) (string, error) {
	extensionBytes, err := json.Marshal(trigger.LogTriggerExtension)
	if err != nil {
		return "", err
	}

	// TODO (auto-4314): Ensure it works with conditionals and add unit tests
	combined := fmt.Sprintf("%s%s", id, extensionBytes)
	hash := crypto.Keccak256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

func (o *Runner) wrapAggregate(r *result[ocr2keepers.CheckResult]) func([]ocr2keepers.CheckResult, error) {
	return func(results []ocr2keepers.CheckResult, err error) {
		if err == nil {
			r.AddSuccesses(1)

			for _, result := range results {
				workID, err := UpkeepWorkID(result.UpkeepID.BigInt(), result.Trigger)
				if err != nil {
					continue
				}

				// only add to the cache if the result is not retryable
				if !result.Retryable {
					o.cache.Set(workID, result, pkgutil.DefaultCacheExpiration)
				}

				r.Add(result)
			}
		} else {
			r.SetErr(err)
			o.logger.Printf("error received from worker result: %s", err)
			r.AddFailures(1)
		}
	}
}
