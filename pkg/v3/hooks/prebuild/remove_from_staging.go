package prebuild

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
)

type resultRemover interface {
	Remove(...string)
}

func NewRemoveFromStaging(remover resultRemover, logger *log.Logger) *RemoveFromStagingHook {
	return &RemoveFromStagingHook{
		remover: remover,
		logger:  log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:remove-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type RemoveFromStagingHook struct {
	remover resultRemover
	logger  *log.Logger
}

func (hook *RemoveFromStagingHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	toRemove := make([]string, 0, len(outcome.Performable))

	for _, result := range outcome.Performable {
		toRemove = append(toRemove, result.Payload.ID)
	}

	hook.logger.Printf("%d results found in outcome", len(toRemove))
	hook.remover.Remove(toRemove...)

	return nil
}
