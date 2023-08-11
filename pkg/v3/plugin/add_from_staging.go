package plugin

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

//go:generate mockery --name ResultViewer --structname MockResultViewer --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin" --case underscore --filename resultviewer.generated.go
type ResultViewer interface {
	View(...ocr2keepersv3.ViewOpt) ([]ocr2keepers.CheckResult, error)
}

func NewAddFromStagingHook(store ocr2keepers.ResultStore, logger *log.Logger, coord Coordinator) AddFromStagingHook {
	return AddFromStagingHook{
		store:  store,
		coord:  coord,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type AddFromStagingHook struct {
	store  ocr2keepers.ResultStore
	logger *log.Logger
	coord  Coordinator
}

func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	results, err := hook.store.View()
	if err != nil {
		return err
	}
	results, err = hook.coord.FilterResults(results)
	if err != nil {
		return err
	}
	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(results), func(i, j int) {
		results[i], results[j] = results[j], results[i]
	})

	// take first limit
	if len(results) > limit {
		results = results[:limit]
	}

	hook.logger.Printf("adding %d results to observation", len(results))
	obs.Performable = append(obs.Performable, results...)

	return nil
}
