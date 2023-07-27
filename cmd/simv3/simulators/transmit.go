package simulators

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

// Transmit sends the report to the on-chain OCR2Aggregator smart
// contract's Transmit method.
//
// In most cases, implementations of this function should store the
// transmission in a queue/database/..., but perform the actual
// transmission (and potentially confirmation) of the transaction
// asynchronously.
func (ct *SimulatedContract) Transmit(
	ctx context.Context,
	digest types.ConfigDigest,
	v uint64,
	r ocr3types.ReportWithInfo[plugin.AutomationReportInfo],
	s []types.AttributedOnchainSignature,
) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// TODO: simulate gas bumping
	return ct.transmitter.Transmit(ct.account, []byte(r.Report), 0, 0)
}

// LatestConfigDigestAndEpoch returns the logically latest configDigest and
// epoch for which a report was successfully transmitted.
func (ct *SimulatedContract) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	err error,
) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	conf, ok := ct.runConfigs[ct.lastConfig.String()]
	if ok {
		return conf.ConfigDigest, ct.lastEpoch, nil
	}

	return types.ConfigDigest{}, 1, fmt.Errorf("config not found")
}

// Account from which the transmitter invokes the contract
func (ct *SimulatedContract) FromAccount() (types.Account, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return types.Account(ct.account), nil
}
