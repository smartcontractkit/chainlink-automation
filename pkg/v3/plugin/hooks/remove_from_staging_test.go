package hooks

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestRemoveFromStagingHook(t *testing.T) {
	tests := []struct {
		Name  string
		Input []ocr2keepers.CheckResult
	}{
		{
			Name:  "No Results",
			Input: []ocr2keepers.CheckResult{},
		},
		{
			Name: "One Result",
			Input: []ocr2keepers.CheckResult{
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
					WorkID:   "1",
				},
			},
		},
		{
			Name: "Five Results",
			Input: []ocr2keepers.CheckResult{
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
					WorkID:   "2",
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
					WorkID:   "3",
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
					WorkID:   "4",
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{4}),
					WorkID:   "5",
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{5}),
					WorkID:   "6",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ob := ocr2keepersv3.AutomationOutcome{
				AgreedPerformables: test.Input,
			}

			mr := &mockResultStore{}

			r := NewRemoveFromStagingHook(mr, log.New(io.Discard, "", 0))

			r.RunHook(ob)
			assert.Equal(t, len(ob.AgreedPerformables), len(mr.removedIDs))
		})
	}
}

// MockResultStore is a mock implementation of types.ResultStore for testing
type mockResultStore struct {
	removedIDs []string
}

func (m *mockResultStore) Remove(ids ...string) {
	m.removedIDs = append(m.removedIDs, ids...)
}

func (m *mockResultStore) Add(...types.CheckResult) {
}

func (m *mockResultStore) View() ([]types.CheckResult, error) {
	return nil, nil
}
