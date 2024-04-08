package runner

import (
	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"sync"
)

type result struct {
	// this struct type isn't expressly defined to run in a single thread or
	// multiple threads so internally a mutex provides the thread safety
	// guarantees in the case it is used in a multi-threaded way
	mu        sync.RWMutex
	successes int
	failures  int
	err       error
	values    []ocr2keepers.CheckResult
}

func newResult() *result {
	return &result{
		values: make([]ocr2keepers.CheckResult, 0),
	}
}

func (r *result) Successes() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.successes
}

func (r *result) AddSuccesses(v int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.successes += v
}

func (r *result) Failures() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.failures
}

func (r *result) AddFailures(v int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failures += v
}

func (r *result) Err() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.err
}

func (r *result) SetErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.err = err
}

func (r *result) Total() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.successes + r.failures
}

func (r *result) unsafeTotal() int {
	return r.successes + r.failures
}

func (r *result) SuccessRate() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.unsafeTotal() == 0 {
		return 0
	}

	return float64(r.successes) / float64(r.unsafeTotal())
}

func (r *result) FailureRate() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.unsafeTotal() == 0 {
		return 0
	}

	return float64(r.failures) / float64(r.unsafeTotal())
}

func (r *result) Add(res ocr2keepers.CheckResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.values = append(r.values, res)
}

func (r *result) Values() []ocr2keepers.CheckResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.values
}
