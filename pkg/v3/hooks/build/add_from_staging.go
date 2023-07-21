package build

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
)

func NewAddFromStaging(rStore ocr2keepersv3.ResultStore, logger *log.Logger) *AddFromStaging {
	return &AddFromStaging{
		rStore: rStore,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
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

	if len(results) > 0 {
		hook.logger.Printf("adding %d results to observation", len(results))

		obs.Performable = append(obs.Performable, results...)
	}

	return nil
}
