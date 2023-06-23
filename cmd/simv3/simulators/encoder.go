package simulators

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

var (
	ErrUnexpectedResult = fmt.Errorf("unexpected result struct")
)

type SimulatedReportEncoder struct {
	ReportGasLimit uint64
}

func (re SimulatedReportEncoder) Encode(results ...ocr2keepers.CheckResult) ([]byte, error) {
	return json.Marshal(results)
}

func (re SimulatedReportEncoder) Extract(b []byte) ([]ocr2keepers.ReportedUpkeep, error) {
	var results []ocr2keepers.CheckResult

	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}

	reported := make([]ocr2keepers.ReportedUpkeep, len(results))

	for i, result := range results {
		reported[i] = ocr2keepers.ReportedUpkeep{
			ID:          result.Payload.ID,
			UpkeepID:    result.Payload.Upkeep.ID,
			Trigger:     result.Payload.Trigger,
			PerformData: result.PerformData,
		}
	}

	return reported, nil
}