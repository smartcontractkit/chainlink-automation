package types

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type UpkeepIdentifier [32]byte

func (u UpkeepIdentifier) String() string {
	return hexutil.Encode(u[:])
}

func (u UpkeepIdentifier) BigInt() *big.Int {
	return big.NewInt(0).SetBytes(u[:])
}

type UpkeepType uint8

// Add exploratory ticket to add default type (0 value) for unknown
const (
	ConditionTrigger UpkeepType = iota
	LogTrigger
)

type TransmitEventType int

const (
	PerformEvent TransmitEventType = iota
	StaleReportEvent
	ReorgReportEvent
	InsufficientFundsReportEvent
)

type BlockNumber uint64

type BlockKey struct {
	Number BlockNumber
	Hash   [32]byte
}

type UpkeepState uint8

const (
	Performed UpkeepState = iota
	Eligible
)

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
	// CheckData is the data used to check the upkeep
	CheckData []byte
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
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

func ValidateTrigger(t Trigger) error {
	if t.BlockNumber == 0 {
		return fmt.Errorf("block number cannot be zero")
	}

	if len(t.BlockHash) == 0 {
		return fmt.Errorf("block hash cannot be empty")
	}

	return nil
}

func NewTrigger(blockNumber BlockNumber, blockHash [32]byte) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (t Trigger) String() string {
	return fmt.Sprintf("%d:%s:%+v", t.BlockNumber, t.BlockHash, t.LogTriggerExtension)
}

// CoordinatedProposal contains all required values to construct a complete
// UpkeepPayload for use in a runner
type CoordinatedProposal struct {
	UpkeepID UpkeepIdentifier
	Trigger  Trigger
}

// Details of an upkeep for which a report was generated
type ReportedUpkeep struct {
	// UpkeepID is the value that identifies a configured upkeep
	UpkeepID UpkeepIdentifier
	// Trigger data for the upkeep
	Trigger Trigger
}

type BlockHistory []BlockKey

func (bh BlockHistory) Latest() (BlockKey, error) {
	if len(bh) == 0 {
		return BlockKey{}, fmt.Errorf("empty block history")
	}

	return bh[0], nil
}

func (bh BlockHistory) Keys() []BlockKey {
	return bh
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
