package plugin

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddBlockHistoryHook struct {
	metadata ocr2keepers.MetadataStore
	logger   *log.Logger
}

func NewAddBlockHistoryHook(ms ocr2keepers.MetadataStore, logger *log.Logger) AddBlockHistoryHook {
	return AddBlockHistoryHook{
		metadata: ms,
		logger:   log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:add-block-history]", telemetry.ServiceName), telemetry.LogPkgStdFlags)}
}

func (h *AddBlockHistoryHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int) error {
	blockHistory := h.metadata.GetBlockHistory()
	if len(blockHistory) > limit {
		blockHistory = blockHistory[:limit]
	}
	obs.BlockHistory = blockHistory
	h.logger.Printf("adding %d blocks to observation", len(blockHistory))
	return nil
}
