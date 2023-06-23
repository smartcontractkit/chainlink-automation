package build

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

func NewAddFromStaging(rStore ocr2keepersv3.ResultStore, logger *log.Logger) *AddFromStaging {
	return &AddFromStaging{rStore: rStore, logger: logger}
}

type AddFromStaging struct {
	rStore ocr2keepersv3.ResultStore
	logger *log.Logger
}

func (hook *AddFromStaging) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	results, err := hook.rStore.View()
	if err != nil {
		return err
	}

	obs.Performable = append(obs.Performable, results...)

	return nil
}
