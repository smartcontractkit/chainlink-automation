package postprocessors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type mockRetryer struct {
	retryCalled bool
}

func (m *mockRetryer) Retry(result ocr2keepers.CheckResult) error {
	m.retryCalled = true
	return nil
}

func TestRetryPostProcessor_PostProcess(t *testing.T) {
	// Create a mock retryer
	retryer := &mockRetryer{}

	// Create a RetryPostProcessor with the mock retryer
	processor := NewRetryPostProcessor(retryer)

	// Create some check results
	results := []ocr2keepers.CheckResult{
		{Retryable: true},
		{Retryable: false},
		{Retryable: true},
	}

	// Call the PostProcess method
	err := processor.PostProcess(context.Background(), results)
	assert.Nil(t, err, "PostProcess returned an error: %v", err)

	// Verify that the Retry method was called for retryable results
	assert.True(t, retryer.retryCalled, "Retry method was not called for retryable results")
}
