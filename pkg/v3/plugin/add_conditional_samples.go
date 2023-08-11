package plugin

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AddConditionalSamplesHook struct {
	metadata types.MetadataStore
	logger   *log.Logger
	coord    types.Coordinator
}

func NewAddConditionalSamplesHook(ms types.MetadataStore, coord types.Coordinator, logger *log.Logger) AddConditionalSamplesHook {
	return AddConditionalSamplesHook{
		metadata: ms,
		coord:    coord,
		logger:   log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-conditional-samples]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

func (h *AddConditionalSamplesHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	conditionals := h.metadata.ViewConditionalProposal()

	var err error
	conditionals, err = h.coord.FilterProposals(conditionals)
	if err != nil {
		return err
	}

	// Shuffle using random seed
	rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(conditionals), func(i, j int) {
		conditionals[i], conditionals[j] = conditionals[j], conditionals[i]
	})

	// take first limit
	if len(conditionals) > limit {
		conditionals = conditionals[:limit]
	}

	h.logger.Printf("adding %d conditional proposals to observation", len(conditionals))
	obs.UpkeepProposals = append(obs.UpkeepProposals, conditionals...)
	return nil
}
