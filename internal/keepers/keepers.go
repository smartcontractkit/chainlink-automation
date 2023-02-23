package keepers

import (
	"context"
	"log"

	"github.com/smartcontractkit/libocr/commontypes"

	"github.com/smartcontractkit/ocr2keepers/internal/malicious"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type MaliciousObservationModifier func(context.Context, []byte, error) (string, []byte, error)
type MaliciousConfigModifier func(malicious.ConfigRemapping) (string, malicious.ConfigRemapping)

type keepers struct {
	id                 commontypes.OracleID
	service            upkeepService
	encoder            types.ReportEncoder
	logger             *log.Logger
	filter             filterer
	reportGasLimit     uint32
	upkeepGasOverhead  uint32
	maxUpkeepBatchSize int
	reportBlockLag     int
	tests              []MaliciousObservationModifier
	configTests        []MaliciousConfigModifier
	selectedTest       int
	timesTested        int
	timesConfigTested  int
}
