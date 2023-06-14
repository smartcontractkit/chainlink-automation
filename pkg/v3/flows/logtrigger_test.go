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
)

func TestLogTriggerEligibilityFlow_EmptySet(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := new(mocks.MockRunner)
	src := new(mocks.MockPreProcessor)
	store := new(mocks.MockResultStore)

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{}, nil).Times(2)

	logFlow := NewLogTriggerEligibility(src, store, runner, logger)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NoError(t, logFlow.Start(context.Background()), "no error expected on start up")
		wg.Done()
	}()

	time.Sleep(2500 * time.Millisecond)

	assert.NoError(t, logFlow.Close(), "no error expected on shut down")

	wg.Wait()
}

func TestLogTriggerEligibilityFlow_SinglePayload(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := new(mocks.MockRunner)
	src := new(mocks.MockPreProcessor)
	store := new(mocks.MockResultStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// will call preprocess on the log source x times and return the same
	// values every time
	src.On("PreProcess", mock.Anything, mock.Anything).
		Return(testData, nil).Times(5)

	runner.On("CheckUpkeeps", mock.Anything, testData).
		Return([]ocr2keepers.CheckResult{
			{Payload: testData[0], Eligible: true},
		}, nil).Times(5)

	store.On("Add", mock.Anything).Times(5)

	logFlow := NewLogTriggerEligibility(src, store, runner, logger)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NoError(t, logFlow.Start(context.Background()), "no error expected on start up")
		wg.Done()
	}()

	time.Sleep(5500 * time.Millisecond)

	assert.NoError(t, logFlow.Close(), "no error expected on shut down")

	runner.AssertExpectations(t)
	src.AssertExpectations(t)
	store.AssertExpectations(t)

	wg.Wait()
}

func TestLogTriggerEligibilityFlow_SingleRetry(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	src := new(mocks.MockPreProcessor)
	store := new(mocks.MockResultStore)

	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}

	// ensure logs preprocessor is called
	src.On("PreProcess", mock.Anything, mock.Anything).Return(testData, nil).Times(1)
	src.On("PreProcess", mock.Anything, mock.Anything).Return(nil, nil).Times(2)

	// within the standard happy path, check upkeeps is called and returns
	// as retryable.
	// the upkeep should be added to the retry path and retried once where the
	// runner again returns retryable.
	/*
		runner.On("CheckUpkeeps", mock.Anything, testData).
			Return([]ocr2keepers.CheckResult{
				{Payload: testData[0], Eligible: false, Retryable: true},
			}, nil).Times(2)

	*/

	// after the first retry returns a retryable result, the upkeep should be
	// retried once more in the runner with the result being eligible
	/*
			runner.On("CheckUpkeeps", mock.Anything, testData).
				Return([]ocr2keepers.CheckResult{
					{Payload: testData[0], Eligible: true, Retryable: false},
				}, nil).Times(1)

		runner.On("CheckUpkeeps", mock.Anything, mock.Anything).
			Return([]ocr2keepers.CheckResult{}, nil).Times(10) // 2 for main 8 for other
	*/

	// after the upkeep is determined to be eligible and not retryable, the
	// result is added to the result store
	store.On("Add", mock.Anything).Times(1)

	logFlow := NewLogTriggerEligibility(src, store, runner, logger)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NoError(t, logFlow.Start(context.Background()), "no error expected on start up")
		wg.Done()
	}()

	time.Sleep(3200 * time.Millisecond)

	assert.NoError(t, logFlow.Close(), "no error expected on shut down")

	src.AssertExpectations(t)
	store.AssertExpectations(t)

	wg.Wait()
}

type mockedRunner struct {
	count         int
	eligibleAfter int
}

func (_m *mockedRunner) CheckUpkeeps(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
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
