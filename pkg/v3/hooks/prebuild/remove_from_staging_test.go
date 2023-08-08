package prebuild

import (
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
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
				},
			},
		},
		{
			Name: "Five Results",
			Input: []ocr2keepers.CheckResult{
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{3}),
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{4}),
				},
				{
					UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{5}),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ob := ocr2keepersv3.AutomationOutcome{
				BasicOutcome: ocr2keepersv3.BasicOutcome{
					Performable: test.Input,
				},
			}

			mr := new(mockRemover)

			r := NewRemoveFromStaging(mr, log.New(io.Discard, "", 0))

			assert.NoError(t, r.RunHook(ob))
			assert.Equal(t, len(ob.Performable), len(mr.removed))
		})
	}
}

type mockRemover struct {
	removed []string
}

func (_m *mockRemover) Remove(toRemove ...string) {
	_m.removed = append(_m.removed, toRemove...)
}
