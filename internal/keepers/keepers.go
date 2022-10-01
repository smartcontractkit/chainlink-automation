package keepers

import (
	"log"
	"math/rand"
	"sync"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type keepers struct {
	id       commontypes.OracleID
	rSrc     rand.Source
	service  upkeepService
	encoder  types.ReportEncoder
	logger   *log.Logger
	mu       sync.Mutex
	transmit bool
}

type observationMessageProto struct {
	RandomValue int64 `json:"random_value"`
	Keys        []types.UpkeepKey
}
