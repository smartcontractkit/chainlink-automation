package keepers

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	service upkeepService
	encoder types.ReportEncoder
}
