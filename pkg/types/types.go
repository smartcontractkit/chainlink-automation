package types

import (
	"context"
	"encoding/json"
	"fmt"
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

type PerformLogProvider interface {
	Subscribe() chan PerformLog
	Unsubscribe()
}

type PerformLog struct {
	Key UpkeepKey
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
	Performed
)

type OffchainConfig struct {
	// PerformLockoutWindow is the window in which a single upkeep cannot be
	// performed again while waiting for a confirmation. Standard setting is
	// 100 blocks * average block time. Units are in milliseconds
	PerformLockoutWindow int64 `json:"performLockoutWindow"`
}

func DecodeOffchainConfig(b []byte) (OffchainConfig, error) {
	var config OffchainConfig
	var err error

	if len(b) > 0 {
		err = json.Unmarshal(b, &config)
	}

	if config.PerformLockoutWindow == 0 {
		config.PerformLockoutWindow = 100 * 12 * 1000 // default of 100 blocks * 12 second blocks
	}

	return config, err
}

func (c OffchainConfig) Encode() []byte {
	b, err := json.Marshal(&c)
	if err != nil {
		panic(fmt.Sprintf("unexpected error json encoding OffChainConfig: %s", err))
	}

	return b
}

/*
type onChainConfig struct {
}
*/
