package hooks

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

func NewBuildHookAddFromStaging() *BuildHookAddFromStaging {
	return &BuildHookAddFromStaging{}
}

type BuildHookAddFromStaging struct{}

func (hook *BuildHookAddFromStaging) RunHook(obs *ocr2keepersv3.AutomationObservation, _ ocr2keepersv3.InstructionStore, _ ocr2keepersv3.MetadataStore, rStore ocr2keepersv3.ResultStore) error {
	results, err := rStore.View()
	if err != nil {
		return err
	}

	obs.Performable = append(obs.Performable, results...)

	return nil
}
