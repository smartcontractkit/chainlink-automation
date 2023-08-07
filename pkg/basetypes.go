package ocr2keepers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Steps
// 1. Split packages for types for v2 and v3 -> ocr2keepers PR
//		In core node, update references. evm20 should point to v2 types and evm21 to v3 types
// 				-> CL PR
// 2. Update v3 types iteratively. Should only affect evm21 and not 20. For every PR, open corresponding Cl PR
//   and verify 21 integration test passes

// Create a new base types package for v3

// Only for v2
type UpkeepIdentifier []byte

// upkeepID is uint256 on contract
type UpkeepIdentifierV3 [256]byte

// Add function to convert to string
// Add function to convert to and from bigInt

type UpkeepType uint8

// Add exploratory ticket to add default type (0 value) for unknown
const (
	ConditionTrigger UpkeepType = iota
	LogTrigger
)

// Only for v2
type BlockKey string

// Only for v2
type UpkeepKey []byte

type BlockNumber uint64
type BlockKeyV3 struct {
	Number BlockNumber
	Hash   [32]byte
}

// Only for v2
type UpkeepResult interface{}

// Only for v2
func upkeepKeysToString(keys []UpkeepKey) string {
	keysStr := make([]string, len(keys))
	for i, key := range keys {
		keysStr[i] = string(key)
	}

	return strings.Join(keysStr, ", ")
}

// Only for v2
type PerformLog struct {
	Key             UpkeepKey
	TransmitBlock   BlockKey
	Confirmations   int64
	TransactionHash string
}

// Only for v2
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

// Only for v3
type TransmitEvent struct {
	// Type describes the type of event
	Type TransmitEventType
	// TransmitBlock is the block height of the transmit event
	TransmitBlock BlockNumber
	// Confirmations is the block height behind latest
	Confirmations int64
	// TransactionHash is the hash for the transaction where the event originated
	TransactionHash [32]byte
	// UpkeepID uniquely identifies the upkeep in the registry
	UpkeepID UpkeepIdentifier
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// CheckBlock is the block value that the upkeep was originally checked at
	CheckBlock BlockNumber
}

// Only for v3
type CheckResult struct {
	// Eligible indicates whether this result is eligible to be performed
	Eligible bool
	// If result is not eligible then the reason it failed
	FailureReason uint8
	// Retryable indicates if this result can be retried on the check pipeline
	Retryable bool
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifierV3
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
	// GasAllocated is the gas to provide an upkeep in a report
	GasAllocated uint64
	// PerformData is the raw data returned when simulating an upkeep perform
	PerformData []byte
	// todo: add comment
	FastGasWei *big.Int
	// todo: add comment
	LinkNative *big.Int
}

// Can be removed
func (r *CheckResult) UnmarshalJSON(b []byte) error {
	type raw struct {
		Eligible     bool
		Retryable    bool
		GasAllocated uint64
		Payload      UpkeepPayload
		PerformData  []byte
		Extension    json.RawMessage
	}

	var basicRaw raw

	if err := json.Unmarshal(b, &basicRaw); err != nil {
		return err
	}

	output := CheckResult{
		Eligible:     basicRaw.Eligible,
		Retryable:    basicRaw.Retryable,
		GasAllocated: basicRaw.GasAllocated,
		Payload:      basicRaw.Payload,
		PerformData:  basicRaw.PerformData,
	}

	if string(basicRaw.Extension) != "null" {
		output.Extension = []byte(basicRaw.Extension)

		var v []byte
		if err := json.Unmarshal(basicRaw.Extension, &v); err == nil {
			output.Extension = v
		}
	}

	*r = output

	return nil
}

func ValidateCheckResult(r CheckResult) error {
	if r.Eligible && r.Retryable {
		return fmt.Errorf("check result cannot be both eligible and retryable")
	}

	if r.GasAllocated == 0 {
		return fmt.Errorf("gas allocated cannot be zero")
	}

	return ValidateUpkeepPayload(r.Payload)
}

type UpkeepPayload struct {
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifierV3
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// CheckData is the data used to check the upkeep
	CheckData []byte
}

type UpkeepTypeGetter func(uid UpkeepIdentifier) UpkeepType

func ValidateUpkeepPayload(p UpkeepPayload) error {
	if len(p.ID) == 0 {
		return fmt.Errorf("upkeep payload id cannot be empty")
	}

	if err := ValidateConfiguredUpkeep(p.Upkeep); err != nil {
		return err
	}

	return ValidateTrigger(p.Trigger)
}

type Trigger struct {
	// BlockNumber is the block number in which the event occurred
	BlockNumber BlockNumber
	// BlockHash is the block hash of the corresponding block
	BlockHash [32]byte
	// Extensions can be different for different triggers
	LogTriggerExtension *LogTriggerExtenstion
}

type LogTriggerExtenstion struct {
	LogTxHash [32]byte
	Index     uint32
}

// Add function on LogTriggerExtenstion to generate log identifier (string)

func ValidateTrigger(t Trigger) error {
	if t.BlockNumber == 0 {
		return fmt.Errorf("block number cannot be zero")
	}

	if len(t.BlockHash) == 0 {
		return fmt.Errorf("block hash cannot be empty")
	}

	return nil
}

func (t *Trigger) UnmarshalJSON(b []byte) error {
	type raw struct {
		BlockNumber int64
		BlockHash   string
		// TODO: consider using map[string]interface{} instead
		Extension json.RawMessage
	}

	var basicRaw raw
	if err := json.Unmarshal(b, &basicRaw); err != nil {
		return err
	}

	output := Trigger{
		BlockNumber: basicRaw.BlockNumber,
		BlockHash:   basicRaw.BlockHash,
	}

	if string(basicRaw.Extension) != "null" {
		output.Extension = []byte(basicRaw.Extension)

		// when decoding the first time, the exension data is set as a byte array
		// of the raw encoded original json. if this is encoded again, it is encoded
		// as a byte array. in that case, decode it into a byte array first before
		// passing the bytes on.
		var v []byte
		if err := json.Unmarshal(basicRaw.Extension, &v); err == nil {
			output.Extension = v
		}
	}

	*t = output

	return nil
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

// CoordinatedProposal contains all required values to construct a complete
// UpkeepPayload for use in a runner
type CoordinatedProposal struct {
	UpkeepID UpkeepIdentifierV3
	Trigger  Trigger
}

// Details of an upkeep for which a report was generated
type ReportedUpkeep struct {
	// UpkeepID is the value that identifies a configured upkeep
	UpkeepID UpkeepIdentifierV3
	// Trigger data for the upkeep
	Trigger Trigger
}

type BlockHistory []BlockKeyV3

func (bh BlockHistory) Latest() (BlockKey, error) {
	if len(bh) == 0 {
		return BlockKey(""), fmt.Errorf("empty block history")
	}

	return bh[0], nil
}

func (bh BlockHistory) Keys() []BlockKey {
	return bh
}

func (bh *BlockHistory) UnmarshalJSON(b []byte) error {
	var raw []string

	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	output := make([]BlockKey, len(raw))
	for i, value := range raw {
		output[i] = BlockKey(value)
	}

	*bh = output

	return nil
}

type UpkeepState uint8

const (
	Performed UpkeepState = iota
	Ineligible
)

// Move interfaces for core components here
type TransmitEventProvider interface {
	TransmitEvents(context.Context) ([]TransmitEvent, error)
}

type ConditionalUpkeepProvider interface {
	GetActiveUpkeeps(context.Context, BlockNumber) ([]UpkeepPayload, error)
}

type PayloadBuilder interface {
	// Can get payloads for a subset of proposals along with an error
	BuildPayloads(context.Context, ...CoordinatedProposal) ([]UpkeepPayload, error)
}

type Runnable interface {
	// Can get results for a subset of payloads along with an error
	CheckUpkeeps(context.Context, ...UpkeepPayload) ([]CheckResult, error)
}

type Encoder interface {
	Encode(...CheckResult) ([]byte, error)
	Extract([]byte) ([]ReportedUpkeep, error)
}

type LogEventProvider interface {
	GetLatestPayloads(context.Context) ([]UpkeepPayload, error)
}

type RecoverableProvider interface {
	GetRecoveryProposals() ([]UpkeepPayload, error)
}
