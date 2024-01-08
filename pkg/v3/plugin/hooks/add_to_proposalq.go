package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewAddToProposalQHook(proposalQ ocr2keepers.ProposalQueue, logger *telemetry.Logger) AddToProposalQHook {
	return AddToProposalQHook{
		proposalQ: proposalQ,
		logger:    telemetry.WrapTelemetryLogger(logger, "pre-build hook:add-to-proposalq"),
	}
}

type AddToProposalQHook struct {
	proposalQ ocr2keepers.ProposalQueue
	logger    *telemetry.Logger
}

func (hook *AddToProposalQHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) {
	addedProposals := 0
	for _, roundProposals := range outcome.SurfacedProposals {
		err := hook.proposalQ.Enqueue(roundProposals...)
		if err != nil {
			// Do not return error, just log and skip this round's proposals
			hook.logger.Printf("Error adding proposals to queue: %v", err)
			continue
		}
		addedProposals += len(roundProposals)
	}
	hook.logger.Printf("Added %d proposals from outcome", addedProposals)

}
