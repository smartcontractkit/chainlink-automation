package hooks

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type resultRemover interface {
	Remove(...string)
}

func NewPrebuildHookRemoveFromStaging(remover resultRemover, logger *log.Logger) *PrebuildHookRemoveFromStaging {
	return &PrebuildHookRemoveFromStaging{remover: remover, logger: logger}
}

type PrebuildHookRemoveFromStaging struct {
	remover resultRemover
	logger  *log.Logger
}

func (hook *PrebuildHookRemoveFromStaging) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	toRemove := make([]string, 0, len(outcome.Performable))

	for _, result := range outcome.Performable {
		toRemove = append(toRemove, result.Payload.ID)
	}

	hook.logger.Printf("%d results found in outcome", len(toRemove))
	hook.remover.Remove(toRemove...)

	return nil
}
