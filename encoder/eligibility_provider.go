package encoder

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type EligibilityProvider interface {
	Eligible(types.UpkeepResult) (bool, error)
}
