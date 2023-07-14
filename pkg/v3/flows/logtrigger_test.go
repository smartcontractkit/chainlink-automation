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
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func TestLogTriggerEligibilityFlow_EmptySet(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("GetLogs", mock.Anything).Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	// metadata store should set the value twice with empty data
	mStore.On("Set", store.ProposalRecoveryMetadata, []ocr2keepers.UpkeepPayload{}).Times(2)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		rStore,
		mStore,
		runner,
		src,
		rec,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			tickers.ScheduleTickerWithDefaults,
		},
	)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(idx int, svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx), "failed to start service at index: %d", idx)
			wg.Done()
		}(i, svcs[i], context.Background())
	}

	// wait long enough for the tickers to run twice
	time.Sleep(110 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	wg.Wait()

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)
}

func TestLogTriggerEligibilityFlow_SinglePayload(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("GetLogs", mock.Anything).
		Return(testData, nil).Times(5)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(5)

	rStore.On("Add", mock.Anything).Times(5)

	// metadata store should set the value 5 times with empty data
	mStore.On("Set", store.ProposalRecoveryMetadata, []ocr2keepers.UpkeepPayload{}).Times(5)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		rStore,
		mStore,
		runner,
		src,
		rec,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			tickers.ScheduleTickerWithDefaults,
		},
	)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	// wait enough time for the tickers to run 5 times
	time.Sleep(260 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	wg.Wait()

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)
}

func TestLogTriggerEligibilityFlow_Retry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// ensure logs preprocessor is called
	src.On("GetLogs", mock.Anything).Return(testData, nil).Times(1)
	src.On("GetLogs", mock.Anything).Return(nil, nil).Times(2)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(3)

	// within the standard happy path, check upkeeps is called and returns
	// as retryable.
	// the upkeep should be added to the retry path and retried once where the
	// runner again returns retryable.

	// after the first retry returns a retryable result, the upkeep should be
	// retried once more in the runner with the result being eligible

	// after the upkeep is determined to be eligible and not retryable, the
	// result is added to the result store
	rStore.On("Add", mock.Anything).Times(1)

	// metadata store should set the value thrice with empty data
	mStore.On("Set", store.ProposalRecoveryMetadata, []ocr2keepers.UpkeepPayload{}).Times(3)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		rStore,
		mStore,
		runner,
		src,
		rec,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
			// set some short time values to confine the tests. the schuduled time that
			// follows should allow the scheduled ticker to retry the provided value on
			// the second tick
			func(c *tickers.ScheduleTickerConfig) {
				c.SendDelay = 30 * time.Millisecond
			},
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			tickers.ScheduleTickerWithDefaults,
		},
	)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(160 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	src.AssertExpectations(t)
	src.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)

	wg.Wait()
}

func TestLogTriggerEligibilityFlow_RecoverFromFailedRetry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// ensure logs preprocessor is called
	src.On("GetLogs", mock.Anything).Return(testData, nil).Times(1)
	src.On("GetLogs", mock.Anything).Return(nil, nil).Times(2)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(3)

	// within the standard happy path, check upkeeps is called and returns
	// as retryable.
	// the upkeep should be added to the retry path and retried once where the
	// runner again returns retryable.

	// after the first retry returns a retryable result, the upkeep should be
	// put in the recoverable path

	// metadata store should set the value once
	mStore.On("Set", store.ProposalRecoveryMetadata, mock.Anything)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		rStore,
		mStore,
		runner,
		src,
		rec,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
			func(c *tickers.ScheduleTickerConfig) {
				c.SendDelay = 30 * time.Millisecond
				// set the max send duration low to force a retry failure
				c.MaxSendDuration = 1 * time.Millisecond
			},
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			tickers.ScheduleTickerWithDefaults,
			// set some short time values to confine the tests. the schuduled time that
			// follows should allow the scheduled ticker to retry the provided value on
			// the second tick
			func(c *tickers.ScheduleTickerConfig) {
				c.SendDelay = 30 * time.Millisecond
			},
		},
	)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(160 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)

	wg.Wait()
}

func TestLogTriggerEligibilityFlow_RecoverFromProvider(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// ensure logs preprocessor is called
	src.On("GetLogs", mock.Anything).Return(nil, nil).Times(4)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return(testData, nil).Times(1)
	rec.On("GetRecoverables").Return(nil, nil).Times(3)

	// within the standard happy path, check upkeeps is called and returns
	// as retryable.
	// the upkeep should be added to the retry path and retried once where the
	// runner again returns retryable.

	// after the first retry returns a retryable result, the upkeep should be
	// put in the recoverable path

	// metadata store should set the value once
	mStore.On("Set", store.ProposalRecoveryMetadata, []ocr2keepers.UpkeepPayload{}).Times(3)
	mStore.On("Set", store.ProposalRecoveryMetadata, testData).Times(1)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		rStore,
		mStore,
		runner,
		src,
		rec,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
		},
		[]tickers.ScheduleTickerConfigFunc{ // recovery configs
			tickers.ScheduleTickerWithDefaults,
			// set some short time values to confine the tests. the schuduled time that
			// follows should allow the scheduled ticker to retry the provided value on
			// the second tick
			func(c *tickers.ScheduleTickerConfig) {
				c.SendDelay = 30 * time.Millisecond
			},
		},
	)

	var wg sync.WaitGroup

	for i := range svcs {
		wg.Add(1)
		go func(svc service.Recoverable, ctx context.Context) {
			assert.NoError(t, svc.Start(ctx))
			wg.Done()
		}(svcs[i], context.Background())
	}

	time.Sleep(210 * time.Millisecond)

	for i := range svcs {
		assert.NoError(t, svcs[i].Close(), "no error expected on shut down")
	}

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)

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
