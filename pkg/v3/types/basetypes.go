package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
)

type UpkeepType uint8

const (
	// Exploratory AUTO 4335: add type for unknown
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
	return u.BigInt().String()
}

func (u UpkeepIdentifier) BigInt() *big.Int {
	i, _ := big.NewInt(0).SetString(hex.EncodeToString(u[:]), 16)
	return i
}

func (u *UpkeepIdentifier) FromBigInt(i *big.Int) {
	copy(u[:], i.Bytes())
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
	// WorkID represents the unit of work for the check result
	WorkID string
}

func (r CheckResult) UniqueID() string {
	var resultBytes []byte

	resultBytes = append(resultBytes, r.FastGasWei.Bytes()...)
	resultBytes = append(resultBytes, big.NewInt(int64(r.GasAllocated)).Bytes()...)
	resultBytes = append(resultBytes, r.LinkNative.Bytes()...)
	resultBytes = append(resultBytes, r.PerformData[:]...)
	resultBytes = append(resultBytes, r.UpkeepID[:]...)
	resultBytes = append(resultBytes, r.Trigger.BlockHash[:]...)
	resultBytes = append(resultBytes, big.NewInt(int64(r.Trigger.BlockNumber)).Bytes()...)
	if r.Trigger.LogTriggerExtension != nil {
		resultBytes = append(resultBytes, []byte(fmt.Sprintf("%+v", r.Trigger.LogTriggerExtension))...)
	}

	return fmt.Sprintf("%x", resultBytes)
}

func (r CheckResult) Validate() error {
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
	// WorkID represents the unit of work for the reported upkeep
	WorkID string
}
