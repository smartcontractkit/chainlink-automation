package types

import "context"

type Registry interface {
	GetActiveUpkeepKeys(context.Context, BlockKey) ([]UpkeepKey, error)
	CheckUpkeep(context.Context, UpkeepKey) (bool, UpkeepResult, error)
}

type ReportEncoder interface {
	EncodeReport([]UpkeepResult) ([]byte, error)
}

type BlockKey string

type Address []byte

type UpkeepKey []byte

type UpkeepResult struct {
	Key         UpkeepKey
	State       UpkeepState
	PerformData []byte
}

type UpkeepState uint
