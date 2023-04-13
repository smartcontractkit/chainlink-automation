package keepers

import (
	"log"

	"github.com/smartcontractkit/libocr/commontypes"

	"github.com/smartcontractkit/ocr2keepers/pkg/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/observer"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	id                 commontypes.OracleID
	service            upkeepService
	observers          []observer.Observer // replaces the service above
	encoder            types.ReportEncoder
	logger             *log.Logger
	coordinator        coordinator.Coordinator
	reportGasLimit     uint32
	upkeepGasOverhead  uint32
	maxUpkeepBatchSize int
	reportBlockLag     int
}
