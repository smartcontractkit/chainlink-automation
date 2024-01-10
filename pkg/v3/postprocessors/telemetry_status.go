package postprocessors

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

func (p *TelemetryStatus) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	for _, payload := range results {
		if err := p.logger.Collect(payload.WorkID, uint64(payload.Trigger.BlockNumber), p.status); err != nil {
			p.logger.Println(err)
		}
	}

	return nil
}
