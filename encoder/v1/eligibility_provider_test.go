package v1

import (
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestEligibilityProvider_Eligible(t *testing.T) {
	for _, tc := range []struct {
		name         string
		upkeepResult types.UpkeepResult

		wantEligible bool
		wantErr      error
	}{
		{
			name: "eligible state on upkeep result is deemed eligible",
			upkeepResult: types.UpkeepResult{
				State: types.Eligible,
			},
			wantEligible: true,
		},
		{
			name: "not eligible state on upkeep result is deemed ineligible",
			upkeepResult: types.UpkeepResult{
				State: types.NotEligible,
			},
			wantEligible: false,
		},
		{
			name: "unknown state on upkeep result is deemed ineligible",
			upkeepResult: types.UpkeepResult{
				State: 123,
			},
			wantEligible: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder()
			isEligible, err := e.Eligible(tc.upkeepResult)
			if isEligible != tc.wantEligible {
				t.Fatalf("unexpected eligibility, want %T, got %T ", tc.wantEligible, isEligible)
			}
			if tc.wantErr != nil && tc.wantErr.Error() != err.Error() {
				t.Fatalf("unexpected error: %s", err.Error())
			}
		})
	}
}
