package keepers

import (
	"log"
	"math/rand"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	id      commontypes.OracleID
	rSrc    rand.Source
	service upkeepService
	encoder types.ReportEncoder
	logger  *log.Logger
	filter  filterer
}
