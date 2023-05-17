package simulators

import (
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type SimulatedReportEncoder struct {
	ReportGasLimit uint64
}

// Eligible returns whether an upkeep result is eligible
func (re SimulatedReportEncoder) Eligible(result ocr2keepers.UpkeepResult) (bool, error) {
	res, ok := result.(SimulatedResult)
	if !ok {
		return false, fmt.Errorf("parse error")
	}

	return res.Eligible, nil
}

// Eligible returns whether an upkeep result is eligible
func (re SimulatedReportEncoder) Detail(result ocr2keepers.UpkeepResult) (ocr2keepers.UpkeepKey, uint32, error) {
	res, ok := result.(SimulatedResult)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected result struct")
	}

	return res.Key, res.ExecuteGas, nil
}

func (re SimulatedReportEncoder) KeysFromReport(b []byte) ([]ocr2keepers.UpkeepKey, error) {
	var results []SimulatedResult

	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}

	keys := make([]ocr2keepers.UpkeepKey, 0, len(results))
	for _, res := range results {
		keys = append(keys, res.Key)
	}

	return keys, nil
}

func (re SimulatedReportEncoder) EncodeReport(toReport []ocr2keepers.UpkeepResult) ([]byte, error) {
	final := make([]SimulatedResult, 0, len(toReport))

	var totalGas uint64

	for _, raw := range toReport {
		res, ok := raw.(SimulatedResult)
		if !ok {
			return nil, fmt.Errorf("unexpected result type")
		}

		totalGas += uint64(res.ExecuteGas)

		if totalGas <= re.ReportGasLimit {
			final = append(final, res)
			continue
		}

		break
	}

	return json.Marshal(final)
}

func (re SimulatedReportEncoder) DecodeReport(b []byte) ([]ocr2keepers.UpkeepResult, error) {
	var report []SimulatedResult

	if err := json.Unmarshal(b, &report); err != nil {
		return nil, err
	}

	output := make([]ocr2keepers.UpkeepResult, len(report))
	for i, res := range report {
		output[i] = res
	}

	return output, nil
}
