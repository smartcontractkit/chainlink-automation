package build

import (
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

func NewBuildHookAddFromStaging(rStore ocr2keepersv3.ResultStore, logger *log.Logger) *BuildHookAddFromStaging {
	return &BuildHookAddFromStaging{rStore: rStore, logger: logger}
}

type BuildHookAddFromStaging struct {
	rStore ocr2keepersv3.ResultStore
	logger *log.Logger
}

func (hook *BuildHookAddFromStaging) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	results, err := hook.rStore.View()
	if err != nil {
		return err
	}

	obs.Performable = append(obs.Performable, results...)

	return nil
}
