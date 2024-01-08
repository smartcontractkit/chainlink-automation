package hooks

import (
	"math/rand"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type AddConditionalProposalsHook struct {
	metadata types.MetadataStore
	logger   *telemetry.Logger
	coord    types.Coordinator
}

func NewAddConditionalProposalsHook(ms types.MetadataStore, coord types.Coordinator, logger *telemetry.Logger) AddConditionalProposalsHook {
	return AddConditionalProposalsHook{
		metadata: ms,
		coord:    coord,
		logger:   telemetry.WrapTelemetryLogger(logger, "build hook:add-conditional-samples"),
	}
}

func (h *AddConditionalProposalsHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	conditionals := h.metadata.ViewProposals(types.ConditionTrigger)

	var err error
	conditionals, err = h.coord.FilterProposals(conditionals)
	if err != nil {
		return err
	}

	// Do random shuffling. Sorting isn't done here as we don't require multiple nodes
	// to agree on the same proposal, hence each node just sends a random subset of its proposals
	rand.New(random.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(conditionals), func(i, j int) {
		conditionals[i], conditionals[j] = conditionals[j], conditionals[i]
	})

	// take first limit
	if len(conditionals) > limit {
		conditionals = conditionals[:limit]
	}

	for _, proposal := range conditionals {
		if err := h.logger.Collect(proposal.WorkID, uint64(proposal.Trigger.BlockNumber), telemetry.Proposed); err != nil {
			h.logger.Println(err.Error())
		}
	}

	h.logger.Printf("adding %d conditional proposals to observation", len(conditionals))
	obs.UpkeepProposals = append(obs.UpkeepProposals, conditionals...)
	return nil
}
