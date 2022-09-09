package keepers

import (
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	service UpkeepService
	encoder types.ReportEncoder
}
