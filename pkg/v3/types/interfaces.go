package types

import "context"

type UpkeepTypeGetter func(uid UpkeepIdentifier) UpkeepType

//go:generate mockery --name Encoder --structname MockEncoder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename encoder.generated.go
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
