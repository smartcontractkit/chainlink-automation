package postprocessors

import (
	"context"
	"testing"
)

type mockRetryer struct {
	retryCalled bool
}

func (m *mockRetryer) Retry(result CheckResult) {
	m.retryCalled = true
}

func TestRetryPostProcessor_PostProcess(t *testing.T) {
	// Create a mock retryer
	retryer := &mockRetryer{}

	// Create a RetryPostProcessor with the mock retryer
	processor := NewRetryPostProcessor(retryer)

	// Create some check results
	results := []CheckResult{
		{Retryable: true},
		{Retryable: false},
		{Retryable: true},
	}

	// Call the PostProcess method
	err := processor.PostProcess(context.Background(), results)
	if err != nil {
		t.Errorf("PostProcess returned an error: %v", err)
	}

	// Verify that the Retry method was called for retryable results
	if !retryer.retryCalled {
		t.Error("Retry method was not called for retryable results")
	}
}
