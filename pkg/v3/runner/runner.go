package runner

import (
	"context"
	"fmt"
	v22 "github.com/smartcontractkit/chainlink-automation/internal/util"
	"github.com/smartcontractkit/chainlink-automation/pkg/util/v3"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
)

const WorkerBatchLimit int = 10

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

// ensure that the runner implements the same interface it consumes to indicate
// the runner simply wraps the underlying runnable with extra features
var _ types.Runnable = &Runner{}

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
	runnable types.Runnable
	// initialized by the constructor
	workers *v3.WorkerGroup                    // parallelizer
	cache   *v3.Cache[ocr2keepers.CheckResult] // result cache
	// configurations
	cacheGcInterval time.Duration
	// run state data
	running atomic.Bool
	chClose chan struct{}
}

type RunnerConfig struct {
	// Workers is the maximum number of workers in worker group
	Workers     int
	CacheExpire time.Duration
	CacheClean  time.Duration
}

// NewRunner provides a new configured runner
func NewRunner(
	logger *log.Logger,
	runnable types.Runnable,
	conf RunnerConfig,
) (*Runner, error) {
	return &Runner{
		logger:          log.New(logger.Writer(), fmt.Sprintf("[%s | check-pipeline-runner]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		runnable:        runnable,
		workers:         v3.NewWorkerGroup(conf.Workers),
		cache:           v3.NewCache[ocr2keepers.CheckResult](conf.CacheExpire),
		cacheGcInterval: conf.CacheClean,
		chClose:         make(chan struct{}, 1),
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

	go o.cache.Start(o.cacheGcInterval)

	<-o.chClose

	return nil
}

// Close stops the cache cleaner and the parallel worker process
func (o *Runner) Close() error {
	if !o.running.Load() {
		return fmt.Errorf("not running")
	}

	o.cache.Stop()
	o.workers.Stop()
	o.running.Swap(false)

	o.chClose <- struct{}{}

	return nil
}

func (o *Runner) worker(ctx context.Context, jobs <-chan []ocr2keepers.UpkeepPayload, result *result) {
	for job := range jobs {
		// process the slice
		results, err := o.runnable.CheckUpkeeps(ctx, job...)
		if err == nil {
			result.AddSuccesses(1)

			for _, res := range results {
				// only add to the cache if pipeline was successful
				if res.PipelineExecutionState == 0 {
					c, ok := o.cache.Get(res.WorkID)
					if !ok || res.Trigger.BlockNumber > c.Trigger.BlockNumber {
						// Add to cache if the workID didn't exist before or if we got a result on a higher checkBlockNumber
						o.cache.Set(res.WorkID, res, v3.DefaultCacheExpiration)
					}
				}

				result.Add(res)
			}
		} else {
			result.SetErr(err)
			o.logger.Printf("error received from worker result: %s", err)
			result.AddFailures(1)
		}
	}
}

// parallelCheck should be satisfied by the Runner
func (o *Runner) parallelCheck(ctx context.Context, payloads []ocr2keepers.UpkeepPayload) (*result, error) {
	result := newResult()

	if len(payloads) == 0 {
		return result, nil
	}

	toRun := make([]ocr2keepers.UpkeepPayload, 0, len(payloads))
	for _, payload := range payloads {
		// if workID is in cache for the given trigger blocknum/hash, add to result directly
		if res, ok := o.cache.Get(payload.WorkID); ok &&
			(res.Trigger.BlockNumber == payload.Trigger.BlockNumber) &&
			(res.Trigger.BlockHash == payload.Trigger.BlockHash) {
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

	//// Create batches from the given keys.
	//// Max keyBatchSize items in the batch.
	//v3.RunJobs(
	//	ctx,
	//	o.workers,
	//	v22.Unflatten(toRun, WorkerBatchLimit),
	//	o.wrapWorkerFunc(),
	//	o.wrapAggregate(result),
	//)

	jobBatches := v22.Unflatten(toRun, WorkerBatchLimit)

	jobsToDo := make(chan []ocr2keepers.UpkeepPayload, len(jobBatches))
	//results := make(chan []ocr2keepers.CheckResult, len(jobBatches))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ { // assuming 10 workers
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			o.worker(ctx, jobsToDo, result)
		}(i)
	}

	go func() {
		for _, job := range jobBatches {
			jobsToDo <- job
		}
		close(jobsToDo)
	}()

	wg.Wait()

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
