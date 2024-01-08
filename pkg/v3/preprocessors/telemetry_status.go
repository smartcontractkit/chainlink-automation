package preprocessors

import (
	"context"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewTelemetryStatus(status telemetry.Status, logger *telemetry.Logger) *TelemetryStatus {
	return &TelemetryStatus{
		status: status,
		logger: logger,
	}
}

type TelemetryStatus struct {
	status telemetry.Status
	logger *telemetry.Logger
}

func (p *TelemetryStatus) PreProcess(
	ctx context.Context,
	payloads []ocr2keepers.UpkeepPayload,
) ([]ocr2keepers.UpkeepPayload, error) {
	for _, payload := range payloads {
		if err := p.logger.Collect(payload.WorkID, uint64(payload.Trigger.BlockNumber), p.status); err != nil {
			p.logger.Println(err)
		}
	}

	return payloads, nil
}
