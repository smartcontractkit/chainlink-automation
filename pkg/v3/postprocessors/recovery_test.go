package postprocessors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type mockRecoverer struct {
	recovererCalled bool
}

func (m *mockRecoverer) Recover(result ocr2keepers.CheckResult) error {
	m.recovererCalled = true
	return nil
}

func TestRecoveryPostProcessor_PostProcess(t *testing.T) {
	// Create a mock recoverer
	recoverer := &mockRecoverer{}

	// Create a RecoveryPostProcessor with the mock recoverer
	processor := NewRecoveryPostProcessor(recoverer)

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
	assert.True(t, recoverer.recovererCalled, "Recover method was not called for retryable results")
}
