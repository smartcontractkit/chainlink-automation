package db

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type UpkeepStateDatabase struct {
	state map[string]ocr2keepers.UpkeepState
}

func NewUpkeepStateDatabase() *UpkeepStateDatabase {
	return &UpkeepStateDatabase{
		state: make(map[string]ocr2keepers.UpkeepState),
	}
}

func (usd *UpkeepStateDatabase) SetUpkeepState(_ context.Context, result ocr2keepers.CheckResult, state ocr2keepers.UpkeepState) error {
	usd.state[result.WorkID] = state

	return nil
}
