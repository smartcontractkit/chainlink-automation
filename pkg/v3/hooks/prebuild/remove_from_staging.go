package prebuild

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type resultRemover interface {
	Remove(...string)
}

func NewRemoveFromStaging(remover resultRemover, logger *log.Logger) *RemoveFromStagingHook {
	return &RemoveFromStagingHook{remover: remover, logger: logger}
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
