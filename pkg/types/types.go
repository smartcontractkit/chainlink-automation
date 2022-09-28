package types

import (
	"context"
	"math/big"
)

type Registry interface {
	GetActiveUpkeepKeys(context.Context, BlockKey) ([]UpkeepKey, error)
	CheckUpkeep(context.Context, UpkeepKey) (bool, UpkeepResult, error)
	IdentifierFromKey(UpkeepKey) (UpkeepIdentifier, error)
}

type ReportEncoder interface {
	EncodeReport([]UpkeepResult) ([]byte, error)
}

type BlockKey string

type Address []byte

// UpkeepKey is an identifier of an upkeep at a moment in time, typically an
// upkeep at a block number
type UpkeepKey []byte

// UpkeepIdentifier is an identifier for an active upkeep, typically a big int
type UpkeepIdentifier []byte

type UpkeepResult struct {
	Key              UpkeepKey
	State            UpkeepState
	PerformData      []byte
	FastGasWei       *big.Int
	LinkNative       *big.Int
	CheckBlockNumber uint32
	CheckBlockHash   [32]byte
}

type UpkeepState uint

const (
	Eligible UpkeepState = iota
	Skip
	Perform
	Reported
)

type OffChainConfig struct {
	// PerformLockoutWindow is the window in which a single upkeep cannot be
	// performed again while waiting for a confirmation. Standard setting is
	// 100 blocks * average block time. Units are in milliseconds
	PerformLockoutWindow int64 `json:"performLockoutWindow"`
}

/*
type onChainConfig struct {
}
*/
