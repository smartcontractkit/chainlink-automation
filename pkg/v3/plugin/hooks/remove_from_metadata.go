package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewRemoveFromMetadataHook(ms types.MetadataStore, logger *telemetry.Logger) RemoveFromMetadataHook {
	return RemoveFromMetadataHook{
		ms:     ms,
		logger: telemetry.WrapTelemetryLogger(logger, "pre-build hook:remove-from-metadata"),
	}
}

type RemoveFromMetadataHook struct {
	ms     types.MetadataStore
	logger *telemetry.Logger
}

func (hook *RemoveFromMetadataHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) {
	removed := 0
	for _, round := range outcome.SurfacedProposals {
		for _, proposal := range round {
			hook.ms.RemoveProposals(proposal)
			removed++
		}
	}
	hook.logger.Printf("%d proposals found in outcome for removal", removed)
}
