package upkeep

import (
	"fmt"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/util"
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
	return util.EncodeCheckResultsToReportBytes(results)
}

func (u Util) Extract(b []byte) ([]ocr2keepers.ReportedUpkeep, error) {
	results, err := util.DecodeCheckResultsFromReportBytes(b)
	if err != nil {
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

// GetType returns the upkeep type from an identifier.
func (u Util) GetType(id ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
	return util.GetUpkeepType(id)
}

// GenerateWorkID creates a unique work id from an identifier and trigger.
func (u Util) GenerateWorkID(id ocr2keepers.UpkeepIdentifier, trigger ocr2keepers.Trigger) string {
	return util.UpkeepWorkID(id, trigger)
}
