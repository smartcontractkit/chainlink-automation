package plugin

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
)

func NewRemoveFromMetadataHook(remover resultRemover, logger *log.Logger) RemoveFromMetadataHook {
	return RemoveFromMetadataHook{
		remover: remover,
		logger:  log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:remove-from-metadata]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type RemoveFromMetadataHook struct {
	remover resultRemover
	logger  *log.Logger
}

func (hook *RemoveFromMetadataHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	// Flatten outcome.AgreedProposals and remove them from metadata

	return nil
}
