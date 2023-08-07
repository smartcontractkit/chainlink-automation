package build

import (
	"fmt"
	"log"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

//go:generate mockery --name ResultViewer --structname MockResultViewer --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/hooks/build" --case underscore --filename resultviewer.generated.go
type ResultViewer interface {
	View(...ocr2keepersv3.ViewOpt) ([]ocr2keepers.CheckResult, error)
}

func NewAddFromStaging(store ResultViewer, logger *log.Logger) *AddFromStaging {
	return &AddFromStaging{
		store:  store,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type AddFromStaging struct {
	store  ResultViewer
	logger *log.Logger
}

func (hook *AddFromStaging) RunHook(obs *ocr2keepersv3.AutomationObservation) error {
	results, err := hook.store.View()
	if err != nil {
		return err
	}

	if len(results) > 0 {
		hook.logger.Printf("adding %d results to observation", len(results))

		obs.Performable = append(obs.Performable, results...)
	}

	return nil
}
