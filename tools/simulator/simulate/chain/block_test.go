package chain_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNextState(t *testing.T) {
	tests := []struct {
		pattern string
		err     error
		states  []int
	}{
		{"0", nil, []int{0, 0, 0}},
		{"1", nil, []int{1, 1, 1}},
		{"", nil, []int{0, 0, 0}},
		{"0001000101", nil, []int{0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0, 0, 0, 1}},
	}

	for _, test := range tests {
		manager, err := chain.NewCheckPipelineStateManager(test.pattern)

		require.ErrorIs(t, err, test.err)

		for idx, expected := range test.states {
			assert.Equal(t, expected, manager.GetNextState(), "state should match for %d", idx)
		}
	}
}
