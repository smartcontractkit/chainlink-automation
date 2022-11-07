package simulators

import (
	"context"
	"fmt"
	"math/big"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func (ct *SimulatedContract) LatestConfigDetails(_ context.Context) (uint64, types.ConfigDigest, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.lastConfig == nil {
		return 0, types.ConfigDigest{}, fmt.Errorf("no config found")
	}

	conf, ok := ct.runConfigs[ct.lastConfig.String()]
	if ok {
		return ct.lastConfig.Uint64(), conf.ConfigDigest, nil
	}

	return ct.lastConfig.Uint64(), [32]byte{}, fmt.Errorf("config not available")
}

func (ct *SimulatedContract) LatestConfig(_ context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	bn := big.NewInt(int64(changedInBlock))
	conf, ok := ct.runConfigs[bn.String()]
	if ok {
		return conf, nil
	}

	return types.ContractConfig{}, fmt.Errorf("config not found at %d", changedInBlock)
}
