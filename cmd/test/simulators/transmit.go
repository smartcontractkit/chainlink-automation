package simulators

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
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
	rc types.ReportContext,
	r types.Report,
	s []types.AttributedOnchainSignature,
) error {
	ct.lastEpoch = rc.Epoch
	// TODO: simulate gas bumping
	return ct.src.Transmit([]byte(r), rc.Epoch)
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
	conf, ok := ct.runConfigs[ct.lastConfig.String()]
	if ok {
		c := types.ContractConfig{
			ConfigCount:           uint64(conf.Count),
			Signers:               parseSigners(conf.Signers),
			Transmitters:          parseTransmitters(conf.Transmitters),
			F:                     uint8(conf.F),
			OnchainConfig:         conf.Onchain,
			OffchainConfigVersion: uint64(conf.OffchainVersion),
			OffchainConfig:        conf.Offchain,
		}

		digest, err := ct.dgst.ConfigDigest(c)

		// TODO: eventually the epoch should come from the coordinated contract
		// much like a confirmed transaction
		return digest, ct.lastEpoch, err
	}

	return types.ConfigDigest{}, 0, fmt.Errorf("config not found")
}

// Account from which the transmitter invokes the contract
func (ct *SimulatedContract) FromAccount() types.Account {
	return types.Account(ct.account)
}
