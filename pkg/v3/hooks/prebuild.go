package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type resultRemover interface {
	Remove(...string)
}

var PrebuildHookRemoveFromStaging = func(store resultRemover) func(ocr2keepersv3.AutomationOutcome) error {
	return func(outcome ocr2keepersv3.AutomationOutcome) error {
		toRemove := make([]string, 0, len(outcome.Performable))

		for _, result := range outcome.Performable {
			toRemove = append(toRemove, result.Payload.ID)
		}

		store.Remove(toRemove...)

		return nil
	}
}
