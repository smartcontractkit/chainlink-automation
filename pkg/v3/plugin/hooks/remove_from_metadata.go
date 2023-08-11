package hooks

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func NewRemoveFromMetadataHook(ms types.MetadataStore, upkeepTypeGetter types.UpkeepTypeGetter, logger *log.Logger) RemoveFromMetadataHook {
	return RemoveFromMetadataHook{
		ms:               ms,
		upkeepTypeGetter: upkeepTypeGetter,
		logger:           log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:remove-from-metadata]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type RemoveFromMetadataHook struct {
	ms               types.MetadataStore
	upkeepTypeGetter types.UpkeepTypeGetter
	logger           *log.Logger
}

func (hook *RemoveFromMetadataHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	removed := 0
	for _, round := range outcome.AgreedProposals {
		for _, proposal := range round {
			if hook.upkeepTypeGetter(proposal.UpkeepID) == types.ConditionTrigger {
				hook.ms.RemoveConditionalProposal(proposal)
				removed++
			} else if hook.upkeepTypeGetter(proposal.UpkeepID) == types.LogTrigger {
				hook.ms.RemoveLogRecoveryProposal(proposal)
				removed++
			}
		}
	}
	hook.logger.Printf("%d proposals found in outcome for removal", removed)
	return nil
}
