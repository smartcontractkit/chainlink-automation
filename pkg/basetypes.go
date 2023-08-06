package ocr2keepers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

type UpkeepIdentifier []byte

type UpkeepType uint8

const (
	ConditionTrigger UpkeepType = iota
	LogTrigger
)

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
	// TODO: auto-4245 remove this
	ID string
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
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

type ConfiguredUpkeep struct {
	// ID uniquely identifies the upkeep
	ID UpkeepIdentifier
	// Type is the event type required to initiate the upkeep
	Type int
	// Config is configuration data specific to the type
	Config interface{}
}

func (u *ConfiguredUpkeep) UnmarshalJSON(b []byte) error {
	type raw struct {
		ID     UpkeepIdentifier
		Type   int
		Config json.RawMessage
	}

	var basicRaw raw

	if err := json.Unmarshal(b, &basicRaw); err != nil {
		return err
	}

	output := ConfiguredUpkeep{
		ID:   basicRaw.ID,
		Type: basicRaw.Type,
	}

	if string(basicRaw.Config) != "null" {
		output.Config = []byte(basicRaw.Config)

		var v []byte
		if err := json.Unmarshal(basicRaw.Config, &v); err == nil {
			output.Config = v
		}
	}

	*u = output

	return nil
}

func ValidateConfiguredUpkeep(u ConfiguredUpkeep) error {
	if len(u.ID) == 0 {
		return fmt.Errorf("invalid upkeep identifier")
	}

	return nil
}

type UpkeepPayload struct {
	// TODO: auto-4245 remove this
	ID string
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// Upkeep is all the information that identifies the upkeep
	Upkeep ConfiguredUpkeep
	// CheckBlock: Deprecated
	CheckBlock BlockKey
	// CheckData is the data used to check the upkeep
	CheckData []byte
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
}

type UpkeepPayloadFactory func(uid *big.Int, tp int, block BlockKey, trigger Trigger, checkData []byte) UpkeepPayload
type WorkIDFactory func() []byte
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
	BlockNumber int64
	// BlockHash is the block hash of the corresponding block
	BlockHash string
	// Extension is the extensions' data that can differ between triggers.
	// e.g. for tx hash and log id for log triggers. Log triggers requires this Extension to be a map with all keys and values in string format
	Extension interface{}
}

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
	UpkeepID UpkeepIdentifier
	Trigger  Trigger
	Block    BlockKey
}

type ReportedUpkeep struct {
	// TODO: auto-4245 remove this
	ID string
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
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
	Eligible
)
