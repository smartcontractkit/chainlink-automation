package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type resultRemover interface {
	Remove(...string)
}

func NewPrebuildHookRemoveFromStaging(remover resultRemover) *PrebuildHookRemoveFromStaging {
	return &PrebuildHookRemoveFromStaging{remover: remover}
}

type PrebuildHookRemoveFromStaging struct {
	remover resultRemover
}

func (hook *PrebuildHookRemoveFromStaging) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	toRemove := make([]string, 0, len(outcome.Performable))

	for _, result := range outcome.Performable {
		toRemove = append(toRemove, result.Payload.ID)
	}

	hook.remover.Remove(toRemove...)

	return nil
}
