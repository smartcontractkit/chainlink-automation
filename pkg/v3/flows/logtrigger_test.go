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

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type mockedRunner struct {
	mu            sync.Mutex
	count         int
	eligibleAfter int
}

// Happy path, log trigger flow only
func TestLogTriggerFlow_EmptySet(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	coord := new(mockedPreprocessor)
	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	pb := new(mocks.MockPayloadBuilder)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)
	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	// get logs should run the same number of times as the happy path
	// ticker
	src.On("GetLogs", mock.Anything).Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	// metadata store should set the value twice with empty data
	mStore.On("Get", store.ProposalRecoveryMetadata).Return(ar, true).Times(2)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		coord,
		rStore,
		mStore,
		runner,
		src,
		rec,
		pb,
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

	assert.Equal(t, 8, coord.Calls(), "calls to coordinator as a preprocessor should equal expected")

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)
}

// Happy path, log trigger flow only
func TestLogTriggerEligibilityFlow_SinglePayload(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	coord := new(mockedPreprocessor)
	runner := &mockedRunner{eligibleAfter: 0}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	pb := new(mocks.MockPayloadBuilder)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)
	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	testData := []ocr2keepers.UpkeepPayload{
		{WorkID: "test"},
	}

	// 1 time with test data, 4 times nil
	src.On("GetLogs", mock.Anything).Return(testData, nil).Times(1)
	src.On("GetLogs", mock.Anything).Return(nil, nil).Times(4)

	// get recoverable should run the same number of times as the happy path
	// ticker
	rec.On("GetRecoverables").Return([]ocr2keepers.UpkeepPayload{}, nil).Times(5)

	// only test data will be added to result store, nil will not
	rStore.On("Add", mock.Anything).Times(1)

	// metadata store should set the value 5 times with empty data
	mStore.On("Get", store.ProposalRecoveryMetadata).Return(ar, true).Times(5)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		coord,
		rStore,
		mStore,
		runner,
		src,
		rec,
		pb,
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

	assert.Equal(t, 20, coord.Calls(), "calls to coordinator as a preprocessor should equal expected")

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)
}

func TestLogTriggerEligibilityFlow_Retry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	coord := new(mockedPreprocessor)
	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	pb := new(mocks.MockPayloadBuilder)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)
	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	testData := []ocr2keepers.UpkeepPayload{
		{WorkID: "test"},
	}

	// 1 time with test data, 2 times nil
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
	mStore.On("Get", store.ProposalRecoveryMetadata).Return(ar, true).Times(3)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		coord,
		rStore,
		mStore,
		runner,
		src,
		rec,
		pb,
		tickerInterval,
		tickerInterval,
		logger,
		[]tickers.ScheduleTickerConfigFunc{ // retry configs
			tickers.ScheduleTickerWithDefaults,
			// set some short time values to confine the tests. the scheduled time that
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

	assert.Equal(t, 12, coord.Calls(), "calls to coordinator as a preprocessor should equal expected")

	src.AssertExpectations(t)
	src.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)

	wg.Wait()
}

func TestLogTriggerEligibilityFlow_RecoverFromFailedRetry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	coord := new(mockedPreprocessor)
	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockLogEventProvider)
	rec := new(mocks.MockRecoverableProvider)
	pb := new(mocks.MockPayloadBuilder)
	rStore := new(mocks.MockResultStore)
	mStore := new(mocks.MockMetadataStore)
	ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

	testData := []ocr2keepers.UpkeepPayload{
		{WorkID: "test"},
	}

	// 1 time with test data and 2 times nil
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
	mStore.On("Get", store.ProposalRecoveryMetadata).Return(ar, true).Times(3)

	// set the ticker time lower to reduce the test time
	tickerInterval := 50 * time.Millisecond

	_, svcs := NewLogTriggerEligibility(
		coord,
		rStore,
		mStore,
		runner,
		src,
		rec,
		pb,
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

	assert.Equal(t, 12, coord.Calls(), "calls to coordinator as a preprocessor should equal expected")

	src.AssertExpectations(t)
	rec.AssertExpectations(t)
	rStore.AssertExpectations(t)
	mStore.AssertExpectations(t)

	wg.Wait()
}

func TestProcessOutcome(t *testing.T) {
	t.Run("no values in outcome", func(t *testing.T) {
		pb := new(mocks.MockPayloadBuilder)
		flow := &LogTriggerEligibility{
			builder: pb,
			logger:  log.New(io.Discard, "", 0),
		}

		testOutcome := ocr2keepersv3.AutomationOutcome{}

		assert.NoError(t, flow.ProcessOutcome(testOutcome), "no error from processing outcome")

		pb.AssertExpectations(t)
	})

	t.Run("proposals are added to retryer", func(t *testing.T) {
		recoverer := new(mocks.MockRetryer)
		pb := new(mocks.MockPayloadBuilder)
		ms := new(mocks.MockMetadataStore)

		flow := &LogTriggerEligibility{
			mStore:    ms,
			recoverer: recoverer,
			builder:   pb,
			logger:    log.New(io.Discard, "", 0),
		}

		ar := util.NewCache[ocr2keepers.CoordinatedProposal](util.DefaultCacheExpiration)

		ms.On("Get", store.ProposalRecoveryMetadata).Return(ar, true)

		testOutcome := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Metadata: map[ocr2keepersv3.OutcomeMetadataKey]interface{}{
					ocr2keepersv3.CoordinatedBlockOutcomeKey: ocr2keepers.BlockKey{
						Number: 4,
					},
					ocr2keepersv3.CoordinatedRecoveryProposalKey: []ocr2keepers.CoordinatedProposal{
						{
							UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{5}),
							Trigger: ocr2keepers.Trigger{
								BlockNumber: 10,
								BlockHash:   [32]byte{1},
							},
						},
					},
				},
			},
		}

		expectedProposal := ocr2keepers.CoordinatedProposal{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{5}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 10,
				BlockHash:   [32]byte{1},
			},
		}

		pb.On("BuildPayload", mock.Anything, expectedProposal).Return(ocr2keepers.UpkeepPayload{
			WorkID: "test",
		}, nil)

		recoverer.On("Retry", mock.Anything).Return(nil)

		assert.NoError(t, flow.ProcessOutcome(testOutcome), "no error from processing outcome")

		pb.AssertExpectations(t)
		recoverer.AssertExpectations(t)
	})
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

type mockedPreprocessor struct {
	mu    sync.Mutex
	calls int
}

func (_m *mockedPreprocessor) PreProcess(_ context.Context, b []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	_m.mu.Lock()
	defer _m.mu.Unlock()

	_m.calls++

	return b, nil
}

func (_m *mockedPreprocessor) Calls() int {
	_m.mu.Lock()
	defer _m.mu.Unlock()

	return _m.calls
}
