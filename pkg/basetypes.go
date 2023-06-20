package ocr2keepers

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

type UpkeepIdentifier []byte

type BlockKey string

type UpkeepKey []byte

type UpkeepResult interface{}

func upkeepKeysToString(keys []UpkeepKey) string {
	keysStr := make([]string, len(keys))
	for i, key := range keys {
		keysStr[i] = string(key)
	}

	return strings.Join(keysStr, ", ")
}

type PerformLog struct {
	Key             UpkeepKey
	TransmitBlock   BlockKey
	Confirmations   int64
	TransactionHash string
}

type StaleReportLog struct {
	Key             UpkeepKey
	TransmitBlock   BlockKey
	Confirmations   int64
	TransactionHash string
}

type CheckResult struct {
	Eligible  bool
	Retryable bool
	GasUsed   uint64
	Payload   UpkeepPayload
}

type ConfiguredUpkeep struct {
	// ID uniquely identifies the upkeep
	ID UpkeepIdentifier
	// Type is the event type required to initiate the upkeep
	Type int
	// Config is configuration data specific to the type
	Config interface{}
}

type UpkeepPayload struct {
	// ID uniquely identifies the upkeep payload
	ID string
	// Upkeep is all the information that identifies the upkeep
	Upkeep ConfiguredUpkeep
	// CheckData is the data used to check the upkeep
	CheckData []byte
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
}

func NewUpkeepPayload(uid *big.Int, tp int, trigger Trigger, checkData []byte) UpkeepPayload {
	p := UpkeepPayload{
		Upkeep: ConfiguredUpkeep{
			ID:   UpkeepIdentifier(uid.Bytes()),
			Type: tp,
		},
		Trigger:   trigger,
		CheckData: checkData,
	}
	p.ID = p.GenerateID()
	return p
}

func (p UpkeepPayload) GenerateID() string {
	id := fmt.Sprintf("%s:%s", p.Upkeep.ID, p.Trigger)
	idh := md5.Sum([]byte(id))
	return hex.EncodeToString(idh[:])
}

type Trigger struct {
	// BlockNumber is the block number of the corresponding block
	BlockNumber int64
	// BlockHash is the block hash of the corresponding block
	BlockHash string
	// Extension is the extensions data that can differ between triggers.
	// e.g. for tx hash and log id for log triggers.
	Extension interface{}
}

func NewTrigger(blockNumber int64, blockHash string, extension interface{}) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		Extension:   extension,
	}
}

func (t Trigger) String() string {
	return fmt.Sprintf("%d:%s:%+v", t.BlockNumber, t.BlockHash, t.Extension)
}
