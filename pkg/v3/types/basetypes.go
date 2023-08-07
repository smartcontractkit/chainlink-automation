package types

import (
	"context"
	"fmt"
	"math/big"
)

// Steps
// 1. Split packages for types for v2 and v3 -> ocr2keepers PR
//		In core node, update references. evm20 should point to v2 types and evm21 to v3 types
// 				-> CL PR
// 2. Update v3 types iteratively. Should only affect evm21 and not 20. For every PR, open corresponding Cl PR
//   and verify 21 integration test passes

// Create a new base types package for v3

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

type UpkeepState uint8

const (
	Performed UpkeepState = iota
	Ineligible
)

// upkeepID is uint256 on contract
type UpkeepIdentifier [32]byte

func (u UpkeepIdentifier) String() string {
	return string(u[:])
}

func (u UpkeepIdentifier) BigInt() *big.Int {
	return big.NewInt(0).SetBytes(u[:])
}

type BlockNumber uint64

type BlockKey struct {
	Number BlockNumber
	Hash   [32]byte
}

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
	// If result is not eligible then the reason it failed
	FailureReason uint8
	// Retryable indicates if this result can be retried on the check pipeline
	Retryable bool
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifier
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

func ValidateCheckResult(r CheckResult) error {
	if r.Eligible && r.Retryable {
		return fmt.Errorf("check result cannot be both eligible and retryable")
	}

	if r.GasAllocated == 0 {
		return fmt.Errorf("gas allocated cannot be zero")
	}

	return nil
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

type UpkeepPayload struct {
	// Upkeep is all the information that identifies the upkeep
	UpkeepID UpkeepIdentifier
	// Trigger is the event that triggered the upkeep to be checked
	Trigger Trigger
	// WorkID uniquely identifies the unit of work for the specified upkeep
	WorkID string
	// CheckData is the data used to check the upkeep
	CheckData []byte
}

type UpkeepTypeGetter func(uid UpkeepIdentifier) UpkeepType

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

func NewTrigger(blockNumber BlockNumber, blockHash [32]byte) Trigger {
	return Trigger{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
	}
}

func (t Trigger) String() string {
	return fmt.Sprintf("%d:%s", t.BlockNumber, t.BlockHash)
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
