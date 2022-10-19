package simulators

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
)

type BlockBroadcaster interface {
	Subscribe(bool) (int, chan config.SymBlock)
}

type Digester interface {
	ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error)
}

type SimulatedContract struct {
	src        BlockBroadcaster
	dgst       Digester
	blocks     map[string]config.SymBlock
	configs    map[string]config.Config
	lastBlock  *big.Int
	lastConfig *big.Int
	notify     chan struct{}
	start      sync.Once
	done       chan struct{}
}

func NewSimulatedContract(src BlockBroadcaster, d Digester) *SimulatedContract {
	return &SimulatedContract{
		src:     src,
		dgst:    d,
		blocks:  make(map[string]config.SymBlock),
		configs: make(map[string]config.Config),
		notify:  make(chan struct{}, 1000),
		done:    make(chan struct{}),
	}
}

func (ct *SimulatedContract) Notify() <-chan struct{} {
	return ct.notify
}

func (ct *SimulatedContract) LatestConfigDetails(_ context.Context) (uint64, types.ConfigDigest, error) {
	//config := ct.configs[ct.lastConfig.String()]
	return ct.lastConfig.Uint64(), [32]byte{}, nil
}

func (ct *SimulatedContract) LatestConfig(_ context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	bn := big.NewInt(int64(changedInBlock))
	conf, ok := ct.configs[bn.String()]
	if ok {
		c := types.ContractConfig{
			ConfigDigest:          types.ConfigDigest{},
			ConfigCount:           uint64(conf.Count),
			Signers:               parseSigners(conf.Signers),
			Transmitters:          parseTransmitters(conf.Transmitters),
			F:                     uint8(conf.F),
			OnchainConfig:         conf.Onchain,
			OffchainConfigVersion: uint64(conf.OffchainVersion),
			OffchainConfig:        conf.Offchain,
		}

		digest, _ := ct.dgst.ConfigDigest(c)
		c.ConfigDigest = digest

		return c, nil
	}

	return types.ContractConfig{}, fmt.Errorf("config not found at %d", changedInBlock)
}

func (ct *SimulatedContract) LatestBlockHeight(_ context.Context) (uint64, error) {
	return ct.lastBlock.Uint64(), nil
}

func (ct *SimulatedContract) run() {
	_, chBlocks := ct.src.Subscribe(true)

	for {
		select {
		case block := <-chBlocks:
			ct.blocks[block.BlockNumber.String()] = block
			ct.lastBlock = block.BlockNumber
			_, ok := ct.configs[block.BlockNumber.String()]
			if ok {
				ct.lastConfig = block.BlockNumber
			}

			go func() { ct.notify <- struct{}{} }()
		case <-ct.done:
			return
		}
	}
}

func (ct *SimulatedContract) Start() {
	ct.start.Do(func() {
		go ct.run()
	})
}

func (ct *SimulatedContract) Stop() {
	close(ct.done)
}

func parseSigners(b [][]byte) []types.OnchainPublicKey {
	out := make([]types.OnchainPublicKey, len(b))
	for i, val := range b {
		out[i] = types.OnchainPublicKey(val)
	}
	return out
}

func parseTransmitters(b []string) []types.Account {
	out := make([]types.Account, len(b))
	for i, val := range b {
		out[i] = types.Account(val)
	}
	return out
}
