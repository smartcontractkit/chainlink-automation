package postprocessors

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func TestRetryPostProcessor_PostProcess(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		// Create a mock retryer
		retryer := new(mockRetryer)
		recoverer := new(mockRetryer)

		// Create a RetryPostProcessor with the mock retryer
		processor := NewRetryPostProcessor(retryer, recoverer)

		// Create some check results
		results := []ocr2keepers.CheckResult{
			{Retryable: true},
			{Retryable: false},
			{Retryable: true},
		}

		retryer.On("Retry", results[0]).Return(nil)
		retryer.On("Retry", results[2]).Return(nil)

		// Call the PostProcess method
		err := processor.PostProcess(context.Background(), results)
		assert.Nil(t, err, "PostProcess returned an error: %v", err)

		// Verify that the Retry method was called for retryable results
		retryer.AssertExpectations(t)
		recoverer.AssertExpectations(t)
	})

	t.Run("retry error; bump to recoverer", func(t *testing.T) {
		// Create a mock retryer with error
		retryer := new(mockRetryer)
		recoverer := new(mockRetryer)

		// Create a RetryPostProcessor with the mock retryer
		processor := NewRetryPostProcessor(retryer, recoverer)

		// Create some check results
		results := []ocr2keepers.CheckResult{
			{Retryable: true},
			{Retryable: false},
			{Retryable: true},
		}

		retryer.On("Retry", results[0]).Return(tickers.ErrSendDurationExceeded)
		retryer.On("Retry", results[2]).Return(tickers.ErrSendDurationExceeded)

		recoverer.On("Retry", results[0]).Return(nil)
		recoverer.On("Retry", results[2]).Return(nil)

		// Call the PostProcess method
		err := processor.PostProcess(context.Background(), results)
		assert.Nil(t, err, "PostProcess returned an error: %v", err)

		// Verify that the Retry method was called for retryable results
		retryer.AssertExpectations(t)
		recoverer.AssertExpectations(t)
	})
	t.Run("unexpected retry error; early exit", func(t *testing.T) {
		// Create a mock retryer with error
		retryer := new(mockRetryer)
		recoverer := new(mockRetryer)

		// Create a RetryPostProcessor with the mock retryer
		processor := NewRetryPostProcessor(retryer, recoverer)

		// Create some check results
		results := []ocr2keepers.CheckResult{
			{Retryable: true},
			{Retryable: false},
			{Retryable: true},
		}

		expectedErr := fmt.Errorf("unhandled error")

		retryer.On("Retry", results[0]).Return(expectedErr)
		retryer.On("Retry", results[2]).Return(expectedErr)

		// Call the PostProcess method
		err := processor.PostProcess(context.Background(), results)
		assert.ErrorIs(t, err, expectedErr, "PostProcess returned error should be %s but was %s", expectedErr, err)

		// Verify that the Retry method was called for retryable results
		retryer.AssertExpectations(t)
		recoverer.AssertExpectations(t)
	})

	t.Run("retries and recovery error", func(t *testing.T) {
		// Create a mock retryer with error
		retryer := new(mockRetryer)
		recoverer := new(mockRetryer)

		// Create a RetryPostProcessor with the mock retryer
		processor := NewRetryPostProcessor(retryer, recoverer)

		// Create some check results
		results := []ocr2keepers.CheckResult{
			{Retryable: true},
			{Retryable: false},
			{Retryable: true},
		}

		retryer.On("Retry", results[0]).Return(tickers.ErrSendDurationExceeded)
		retryer.On("Retry", results[2]).Return(tickers.ErrSendDurationExceeded)

		recoverer.On("Retry", results[0]).Return(tickers.ErrSendDurationExceeded)
		recoverer.On("Retry", results[2]).Return(tickers.ErrSendDurationExceeded)

		// Call the PostProcess method
		err := processor.PostProcess(context.Background(), results)
		assert.ErrorIs(t, err, tickers.ErrSendDurationExceeded, "PostProcess returned error should be: %s but was %s", tickers.ErrSendDurationExceeded, err)

		// Verify that the Retry method was called for retryable results
		retryer.AssertExpectations(t)
		recoverer.AssertExpectations(t)
	})
}

type mockRetryer struct {
	mock.Mock
}

func (_m *mockRetryer) Retry(result ocr2keepers.CheckResult) error {
	res := _m.Called(result)
	return res.Error(0)
}
