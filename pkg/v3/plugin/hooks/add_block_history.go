package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type AddBlockHistoryHook struct {
	metadata ocr2keepers.MetadataStore
	logger   *telemetry.Logger
}

func NewAddBlockHistoryHook(ms ocr2keepers.MetadataStore, logger *telemetry.Logger) AddBlockHistoryHook {
	return AddBlockHistoryHook{
		metadata: ms,
		logger:   telemetry.WrapTelemetryLogger(logger, "build hook:add-block-history"),
	}
}

func (h *AddBlockHistoryHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int) {
	blockHistory := h.metadata.GetBlockHistory()
	if len(blockHistory) > limit {
		blockHistory = blockHistory[:limit]
	}
	obs.BlockHistory = blockHistory
	h.logger.Printf("adding %d blocks to observation", len(blockHistory))
}
