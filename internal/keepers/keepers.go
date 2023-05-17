package keepers

import (
	"log"

	"github.com/smartcontractkit/libocr/commontypes"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	id                 commontypes.OracleID
	service            upkeepService
	observers          []Observer
	encoder            types.ReportEncoder
	logger             *log.Logger
	coordinator        Coordinator
	reportGasLimit     uint32
	upkeepGasOverhead  uint32
	maxUpkeepBatchSize int
	reportBlockLag     int
	mercuryLookup      bool
}
