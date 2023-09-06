package upkeep

import (
	"context"
	"encoding/json"
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

var (
	ErrUnexpectedResult = fmt.Errorf("unexpected result struct")
)

// Util contains basic utilities for upkeeps.
type Util struct {
	ReportGasLimit uint64
}

func (u Util) Encode(results ...ocr2keepers.CheckResult) ([]byte, error) {
	return json.Marshal(results)
}

func (u Util) Extract(b []byte) ([]ocr2keepers.ReportedUpkeep, error) {
	var results []ocr2keepers.CheckResult

	if err := json.Unmarshal(b, &results); err != nil {
		return nil, err
	}

	reported := make([]ocr2keepers.ReportedUpkeep, len(results))

	for i, result := range results {
		reported[i] = ocr2keepers.ReportedUpkeep{
			UpkeepID: result.UpkeepID,
			Trigger:  result.Trigger,
			WorkID:   result.WorkID,
		}
	}

	return reported, nil
}

// BuildPayloads creates payloads from proposals.
func (u Util) BuildPayloads(context.Context, ...ocr2keepers.CoordinatedBlockProposal) ([]ocr2keepers.UpkeepPayload, error) {
	// TODO: consider optionally checking for valid active and performable upkeeps

	return nil, nil
}

// GetType returns the upkeep type from an identifier.
func (u Util) GetType(ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {

	return 0
}

// GenerateWorkID creates a unique work id from an identifier and trigger.
func (u Util) GenerateWorkID(ocr2keepers.UpkeepIdentifier, ocr2keepers.Trigger) string {

	return ""
}
