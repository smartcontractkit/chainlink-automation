package executer

import (
	"sync"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type Result struct {
	// mutex not expressly needed but added for safety
	mu        sync.RWMutex
	successes int
	failures  int
	err       error
	values    []ocr2keepers.UpkeepResult
}

func NewResult() *Result {
	return &Result{
		values: make([]ocr2keepers.UpkeepResult, 0),
	}
}

func (r *Result) Successes() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.successes
}

func (r *Result) AddSuccesses(v int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.successes += v
}

func (r *Result) Failures() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.failures
}

func (r *Result) AddFailures(v int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failures += v
}

func (r *Result) Err() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.err
}

func (r *Result) SetErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.err = err
}

func (r *Result) Total() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.successes + r.failures
}

func (r *Result) unsafeTotal() int {
	return r.successes + r.failures
}

func (r *Result) SuccessRate() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.unsafeTotal() == 0 {
		return 0
	}

	return float64(r.successes) / float64(r.unsafeTotal())
}

func (r *Result) FailureRate() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.unsafeTotal() == 0 {
		return 0
	}

	return float64(r.failures) / float64(r.unsafeTotal())
}

func (r *Result) Add(res ocr2keepers.UpkeepResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.values = append(r.values, res)
}

func (r *Result) Values() []ocr2keepers.UpkeepResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.values
}
