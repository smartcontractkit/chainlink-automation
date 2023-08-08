package types

import "context"

type UpkeepTypeGetter func(uid UpkeepIdentifier) UpkeepType

//go:generate mockery --name Encoder --structname MockEncoder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename encoder.generated.go
type Encoder interface {
	Encode(...CheckResult) ([]byte, error)
	Extract([]byte) ([]ReportedUpkeep, error)
}

//go:generate mockery --name LogEventProvider --structname MockLogEventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename logeventprovider.generated.go
type LogEventProvider interface {
	GetLatestPayloads(context.Context) ([]UpkeepPayload, error)
}

//go:generate mockery --name RecoverableProvider --structname MockRecoverableProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename recoverableprovider.generated.go
type RecoverableProvider interface {
	GetRecoveryProposals() ([]UpkeepPayload, error)
}

type TransmitEventProvider interface {
	TransmitEvents(context.Context) ([]TransmitEvent, error)
}

type ConditionalUpkeepProvider interface {
	GetActiveUpkeeps(context.Context, BlockNumber) ([]UpkeepPayload, error)
}

//go:generate mockery --name PayloadBuilder --structname MockPayloadBuilder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename payloadbuilder.generated.go
type PayloadBuilder interface {
	// Can get payloads for a subset of proposals along with an error
	BuildPayloads(context.Context, ...CoordinatedProposal) ([]UpkeepPayload, error)
}

//go:generate mockery --name Runnable --structname MockRunnable --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/types" --case underscore --filename runnable.generated.go
type Runnable interface {
	// Can get results for a subset of payloads along with an error
	CheckUpkeeps(context.Context, ...UpkeepPayload) ([]CheckResult, error)
}
