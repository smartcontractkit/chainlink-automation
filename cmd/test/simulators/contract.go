package simulators

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type BlockBroadcaster interface {
	Subscribe(bool) (int, chan config.SymBlock)
	Unsubscribe(int)
	Transmit([]byte, uint32) error
}

type Digester interface {
	ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error)
}

type SimulatedContract struct {
	src    BlockBroadcaster
	enc    ktypes.ReportEncoder
	logger *log.Logger
	dgst   Digester
	// blocks come from a simulated block provider. this value is to store
	// the blocks as they come in for reference.
	blocks map[string]config.SymBlock
	// runConfigs are OCR configurations defined in the runbook. the key is the
	// blocknumber that the config is included in the simulated blockchain.
	runConfigs map[string]config.Config
	lastBlock  *big.Int
	// lastConfig is the last blocknumber a config was read from
	lastConfig *big.Int
	// lastEpoch is the last epoch in which a transmit took place
	lastEpoch uint32
	// account is the account that this contract simulates for
	account string
	// block subscription id for unsubscribing from channel
	subscription int
	// upkeep mapping of big int id to simulated upkeep
	upkeeps map[string]*SimulatedUpkeep
	perLogs *sortedKeyMap[[]ktypes.PerformLog]
	notify  chan struct{}
	start   sync.Once
	done    chan struct{}
}

func NewSimulatedContract(src BlockBroadcaster, d Digester, sym []SimulatedUpkeep, enc ktypes.ReportEncoder, l *log.Logger) *SimulatedContract {
	return &SimulatedContract{
		src:        src,
		enc:        enc,
		dgst:       d,
		logger:     l,
		blocks:     make(map[string]config.SymBlock),
		runConfigs: make(map[string]config.Config),
		perLogs:    newSortedKeyMap[[]ktypes.PerformLog](),
		notify:     make(chan struct{}, 1000),
		done:       make(chan struct{}),
	}
}

func (ct *SimulatedContract) Notify() <-chan struct{} {
	return ct.notify
}

func (ct *SimulatedContract) LatestConfigDetails(_ context.Context) (uint64, types.ConfigDigest, error) {
	ct.logger.Printf("latest config and details")
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

		return ct.lastConfig.Uint64(), digest, err
	}

	return ct.lastConfig.Uint64(), [32]byte{}, fmt.Errorf("config not available yet")
}

func (ct *SimulatedContract) LatestConfig(_ context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	bn := big.NewInt(int64(changedInBlock))
	conf, ok := ct.runConfigs[bn.String()]
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
	sub, chBlocks := ct.src.Subscribe(true)
	ct.subscription = sub

	for {
		select {
		case block := <-chBlocks:
			ct.blocks[block.BlockNumber.String()] = block
			ct.lastBlock = block.BlockNumber
			_, ok := ct.runConfigs[block.BlockNumber.String()]
			if ok {
				ct.lastConfig = block.BlockNumber
			}

			if block.LatestEpoch != nil {
				if *block.LatestEpoch > ct.lastEpoch {
					ct.lastEpoch = *block.LatestEpoch
				}

				results, err := ct.enc.DecodeReport(block.TransmittedData)
				if err != nil {
					continue
				}

				logs := make([]ktypes.PerformLog, len(results))
				for i, result := range results {
					logs[i] = ktypes.PerformLog{
						Key:           result.Key,
						TransmitBlock: ktypes.BlockKey([]byte(block.BlockNumber.String())),
						Confirmations: 0,
					}

					id, _ := ct.IdentifierFromKey(result.Key)
					up, ok := ct.upkeeps[string(id)]
					if ok {
						//result.PerformData

					}
				}
				ct.perLogs.Set(block.BlockNumber.String(), logs)
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
	ct.src.Unsubscribe(ct.subscription)
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
