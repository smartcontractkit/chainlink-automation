package plugin

import (
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
)

type AddFromSamplesHook struct {
	metadata *store.Metadata
	coord    Coordinator
}

func NewAddFromSamplesHook(ms *store.Metadata, coord Coordinator) AddFromSamplesHook {
	return AddFromSamplesHook{metadata: ms, coord: coord}
}

func (h *AddFromSamplesHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	// TODO: Read conditional samples from metadata store
	// TODO: filter proposals using coordinator
	// Shuffle using random seed
	/*rand.New(util.NewKeyedCryptoRandSource(rSrc)).Shuffle(len(performable), func(i, j int) {
		performable[i], performable[j] = performable[j], performable[i]
	})*/
	// take first limit
	// TODO: Append to obs.CoordinatedProposals

	return nil
}
