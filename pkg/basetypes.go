package ocr2keepers

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
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

type TransmitEventType int

const (
	PerformEvent TransmitEventType = iota
	StaleReportEvent
	ReorgReportEvent
	InsufficientFundsReportEvent
)

type TransmitEvent struct {
	// Type describes the type of event
	Type TransmitEventType
	// TransmitBlock is the block height of the transmit event
	TransmitBlock BlockKey
	// Confirmations is the block height behind latest
	Confirmations int64
	// TransactionHash is the hash for the transaction where the event originated
	TransactionHash string
	// ID uniquely identifies the upkeep/trigger that created this perform log
	ID string
	// UpkeepID uniquely identifies the upkeep in the registry
	UpkeepID UpkeepIdentifier
	// CheckBlock is the block value that the upkeep was originally checked at
	CheckBlock BlockKey
}

type CheckResult struct {
	// Eligible indicates whether this result is eligible to be performed
	Eligible bool
	// Retryable indicates if this result can be retried on the check pipeline
	Retryable bool
	// GasAllocated is the gas to provide an upkeep in a report
	GasAllocated uint64
	// Payload is the detail used to check the upkeep
	Payload UpkeepPayload
	// PerformData is the raw data returned when simulating an upkeep perform
	PerformData []byte
	// Extension is extra data that can differ between contracts
	Extension interface{}
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
	// CheckBlock
	CheckBlock BlockKey
	// CheckData is the data used to check the upkeep
	CheckData []byte
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
}

func NewUpkeepPayload(uid *big.Int, tp int, block BlockKey, trigger Trigger, checkData []byte) UpkeepPayload {
	p := UpkeepPayload{
		Upkeep: ConfiguredUpkeep{
			ID:   UpkeepIdentifier(uid.Bytes()),
			Type: tp,
		},
		CheckBlock: block,
		Trigger:    trigger,
		CheckData:  checkData,
	}
	p.ID = p.GenerateID()
	return p
}

func (p UpkeepPayload) GenerateID() string {
	id := fmt.Sprintf("%s:%s", p.Upkeep.ID, p.Trigger)
	idh := crypto.Keccak256([]byte(id))
	return hex.EncodeToString(idh[:])
}

type Trigger struct {
	// BlockNumber is the block number in which the event occurred
	BlockNumber int64
	// BlockHash is the block hash of the corresponding block
	BlockHash string
	// Extension is the extensions data that can differ between triggers.
	// e.g. for tx hash and log id for log triggers. Log triggers requires this Extention to be a map with all keys and values in string format
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

type ReportedUpkeep struct {
	// ID uniquely identifies the upkeep in the report
	ID string
	// UpkeepID is the value that identifies a configured upkeep
	UpkeepID UpkeepIdentifier
	// Trigger data for the upkeep
	Trigger Trigger
	// PerformData is the data to perform an upkeep with
	PerformData []byte
}

type BlockHistory []BlockKey

func (bh BlockHistory) Latest() (BlockKey, error) {
	if len(bh) == 0 {
		return BlockKey(""), fmt.Errorf("empty block history")
	}

	return bh[0], nil
}

func (bh BlockHistory) Keys() []BlockKey {
	return bh
}

type UpkeepState uint8

const Performed UpkeepState = iota
