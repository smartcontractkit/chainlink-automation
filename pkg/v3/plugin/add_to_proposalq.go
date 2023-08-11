package plugin

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func NewAddToProposalQ(proposalQ ocr2keepers.ProposalQueue, logger *log.Logger) AddToProposalQHook {
	return AddToProposalQHook{
		proposalQ: proposalQ,
		logger:    log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:add-to-proposalq]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type AddToProposalQHook struct {
	proposalQ ocr2keepers.ProposalQueue
	logger    *log.Logger
}

func (hook *AddToProposalQHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	addedProposals := 0
	for _, roundProposals := range outcome.AgreedProposals {
		hook.proposalQ.Enqueue(roundProposals...)
		addedProposals += len(roundProposals)
	}
	hook.logger.Printf("Added %d proposals from outcome", addedProposals)

	return nil
}
