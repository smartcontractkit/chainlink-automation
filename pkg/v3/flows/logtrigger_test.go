package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func TestLogTriggerEligibilityFlow_EmptySet(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	store := new(mocks.MockResultStore)

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("GetLogs", mock.Anything).Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	_, svcs := NewLogTriggerEligibility(store, runner, src, logger, []tickers.RetryConfigFunc{}, []tickers.RetryConfigFunc{})

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(2500 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	wg.Wait()

	src.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestLogTriggerEligibilityFlow_SinglePayload(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	store := new(mocks.MockResultStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("GetLogs", mock.Anything).
		Return(testData, nil).Times(5)

	store.On("Add", mock.Anything).Times(5)

	_, svcs := NewLogTriggerEligibility(store, runner, src, logger, []tickers.RetryConfigFunc{}, []tickers.RetryConfigFunc{})

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(5500 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	wg.Wait()

	src.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestLogTriggerEligibilityFlow_Retry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	store := new(mocks.MockResultStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// ensure logs preprocessor is called
	src.On("GetLogs", mock.Anything).Return(testData, nil).Times(1)
	src.On("GetLogs", mock.Anything).Return(nil, nil).Times(2)

	// within the standard happy path, check upkeeps is called and returns
	// as retryable.
	// the upkeep should be added to the retry path and retried once where the
	// runner again returns retryable.

	// after the first retry returns a retryable result, the upkeep should be
	// retried once more in the runner with the result being eligible

	// after the upkeep is determined to be eligible and not retryable, the
	// result is added to the result store
	store.On("Add", mock.Anything).Times(1)

	// set some short time values to confine the tests
	config := func(c *tickers.RetryConfig) {
		c.RetryDelay = 500 * time.Millisecond
	}

	retryConfigFuncs := []tickers.RetryConfigFunc{
		tickers.RetryWithDefaults,
		config,
	}

	recoveryConfigFuncs := []tickers.RetryConfigFunc{
		tickers.RecoveryWithDefaults,
		config,
	}

	_, svcs := NewLogTriggerEligibility(store, runner, src, logger, retryConfigFuncs, recoveryConfigFuncs)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(3200 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	src.AssertExpectations(t)
	store.AssertExpectations(t)

	wg.Wait()
}

type mockedRunner struct {
	mu            sync.Mutex
	count         int
	eligibleAfter int
}

func (_m *mockedRunner) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	_m.mu.Lock()
	defer _m.mu.Unlock()

	results := make([]ocr2keepers.CheckResult, 0)

	for i := range payloads {
		_m.count++

		var eligible bool
		if _m.count > _m.eligibleAfter {
			eligible = true
		}

		results = append(results, ocr2keepers.CheckResult{
			Payload:   payloads[i],
			Eligible:  eligible,
			Retryable: !eligible,
		})
	}

	return results, nil
}
