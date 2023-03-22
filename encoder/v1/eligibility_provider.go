package v1

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type eligibilityProvider struct{}

// Eligible returns whether an upkeep result is eligible
func (p eligibilityProvider) Eligible(result types.UpkeepResult) (bool, error) {
	return result.State == types.Eligible, nil
}
