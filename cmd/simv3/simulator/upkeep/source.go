package upkeep

import (
	"context"
	"log"
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// Source maintains delivery of active upkeeps based on type and repeat
// return behavior.
type Source struct {
	Util

	// provided dependencies
	listener  *chain.Listener
	active    *ActiveTracker
	triggered *LogTriggerTracker
	logger    *log.Logger

	// configurations
	recoveryLookback int
}

func NewSource(
	listener *chain.Listener,
	active *ActiveTracker,
	triggered *LogTriggerTracker,
	lookback int,
	logger *log.Logger,
) *Source {
	return &Source{
		listener:         listener,
		active:           active,
		triggered:        triggered,
		logger:           log.New(logger.Writer(), "[upkeep-source]", log.LstdFlags),
		recoveryLookback: lookback,
	}
}

// GetActiveUpkeeps delivers all active conditional upkeep payloads. Payloads
// returned should be at least once and only when active.
func (src *Source) GetActiveUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.active.GetAllByType(chain.ConditionalType)
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))

	for i, upkeep := range upkeeps {
		payloads[i] = makePayloadFromUpkeep(upkeep)
	}

	return payloads, nil
}

// GetLatestPayloads returns payloads that have been triggered by watching logs.
// Each payload is delivered exactly once to the caller.
func (src *Source) GetLatestPayloads(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.triggered.GetOnce()
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))
	for i, upkeep := range upkeeps {
		payloads[i] = makeLogPayloadFromUpkeep(upkeep)
	}

	return payloads, nil
}

// GetRecoveryProposals returns payloads that have been triggered by watched
// logs and have already been returned by GetLatestPayloads but have not been
// completed. Each payload is delivered at least once but may be repeated
// multiple times.
func (src *Source) GetRecoveryProposals(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	upkeeps := src.triggered.GetAfter(new(big.Int).SetInt64(int64(src.recoveryLookback)))
	if len(upkeeps) == 0 {
		return nil, nil
	}

	payloads := make([]ocr2keepers.UpkeepPayload, len(upkeeps))
	for i, upkeep := range upkeeps {
		payloads[i] = makeLogPayloadFromUpkeep(upkeep)
	}

	return payloads, nil
}

// TODO: need to complete
func makePayloadFromUpkeep(upkeep chain.SimulatedUpkeep) ocr2keepers.UpkeepPayload {
	return ocr2keepers.UpkeepPayload{}
}

// TODO: need to complete
func makeLogPayloadFromUpkeep(triggered triggeredUpkeep) ocr2keepers.UpkeepPayload {
	return ocr2keepers.UpkeepPayload{}
}
