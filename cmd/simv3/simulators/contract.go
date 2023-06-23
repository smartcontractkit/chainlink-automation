package simulators

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
)

type BlockBroadcaster interface {
	Subscribe(bool) (int, chan config.SymBlock)
	Unsubscribe(int)
}

type Transmitter interface {
	Transmit(string, []byte, uint32, uint8) error
}

type Digester interface {
	ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error)
}

type ContractTelemetry interface {
	CheckID(string, ocr2keepers.BlockKey)
}

type RPCTelemetry interface {
	RegisterCall(string, time.Duration, error)
	AddRateDataPoint(int)
}

type SimulatedContract struct {
	mu     sync.RWMutex
	src    BlockBroadcaster
	enc    plugin.Encoder
	logger *log.Logger
	dgst   Digester
	// blocks come from a simulated block provider. this value is to store
	// the blocks as they come in for reference.
	blocks map[string]config.SymBlock
	// runConfigs are OCR configurations defined in the runbook. the key is the
	// blocknumber that the config is included in the simulated blockchain.
	runConfigs map[string]types.ContractConfig
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
	upkeeps    map[string]SimulatedUpkeep
	perLogs    *sortedKeyMap[[]ocr2keepers.TransmitEvent]
	avgLatency int
	chHeads    chan ocr2keepers.BlockKey

	telemetry ContractTelemetry

	rpc *SimulatedRPC

	transmitter Transmitter
	notify      chan struct{}
	start       sync.Once
	done        chan struct{}
}

func NewSimulatedContract(
	src BlockBroadcaster,
	d Digester,
	sym []SimulatedUpkeep,
	enc plugin.Encoder,
	transmitter Transmitter,
	avgLatency int,
	account string,
	rpcErrorRate float64,
	rpcLoadLimitThreshold int,
	telemetry ContractTelemetry,
	rpcTelemetry RPCTelemetry,
	l *log.Logger,
) *SimulatedContract {
	upkeeps := make(map[string]SimulatedUpkeep)
	for _, upkeep := range sym {
		upkeep.Performs = make(map[string]ocr2keepers.TransmitEvent)
		upkeeps[upkeep.ID.String()] = upkeep
	}

	rpc := NewSimulatedRPC(rpcErrorRate, rpcLoadLimitThreshold, avgLatency, rpcTelemetry)

	return &SimulatedContract{
		src:         src,
		enc:         enc,
		dgst:        d,
		logger:      l,
		avgLatency:  avgLatency,
		account:     account,
		transmitter: transmitter,
		runConfigs:  make(map[string]types.ContractConfig),
		blocks:      make(map[string]config.SymBlock),
		perLogs:     newSortedKeyMap[[]ocr2keepers.TransmitEvent](),
		upkeeps:     upkeeps,
		chHeads:     make(chan ocr2keepers.BlockKey, 1),
		telemetry:   telemetry,
		rpc:         rpc,
		notify:      make(chan struct{}, 1000),
		done:        make(chan struct{}),
	}
}

func (ct *SimulatedContract) Notify() <-chan struct{} {
	return ct.notify
}

func (ct *SimulatedContract) LatestBlockHeight(_ context.Context) (uint64, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.lastBlock == nil {
		return 0, fmt.Errorf("no config found")
	}
	return ct.lastBlock.Uint64(), nil
}

func (ct *SimulatedContract) run() {
	sub, chBlocks := ct.src.Subscribe(true)
	ct.subscription = sub

	for {
		select {
		case block := <-chBlocks:
			ct.logger.Printf("received block %s", block.BlockNumber)

			ct.mu.Lock()
			ct.blocks[block.BlockNumber.String()] = block
			ct.lastBlock = block.BlockNumber

			if block.Config != nil {
				ct.logger.Printf("new config identified at block: %s", block.BlockNumber)
				ct.lastConfig = block.BlockNumber
				ct.runConfigs[block.BlockNumber.String()] = *block.Config
			}

			if block.LatestEpoch != nil {
				if *block.LatestEpoch > ct.lastEpoch {
					ct.lastEpoch = *block.LatestEpoch
				}

				for _, b := range block.TransmittedData {
					reported, err := ct.enc.Extract(b)
					if err != nil {
						continue
					}

					logs := make([]ocr2keepers.TransmitEvent, len(reported))
					for i, result := range reported {
						logs[i] = ocr2keepers.TransmitEvent{
							Type:            ocr2keepers.PerformEvent,
							TransmitBlock:   ocr2keepers.BlockKey(block.BlockNumber.String()),
							Confirmations:   0,
							TransactionHash: "",
							ID:              result.ID,
							UpkeepID:        result.UpkeepID,
							CheckBlock:      ocr2keepers.BlockKey("1"), // TODO: need to get this from somewhere
						}

						up, ok := ct.upkeeps[string(result.UpkeepID)]
						if ok {
							//result.PerformData
							up.Performs[block.BlockNumber.String()] = logs[i]
						}

						ct.logger.Printf("log for key '%s' found in block '%s'", result.ID, block.BlockNumber)
					}

					lgs, ok := ct.perLogs.Get(block.BlockNumber.String())
					if !ok {
						ct.perLogs.Set(block.BlockNumber.String(), logs)
					} else {
						ct.perLogs.Set(block.BlockNumber.String(), append(lgs, logs...))
					}
				}
			}
			ct.mu.Unlock()
			go func() { ct.notify <- struct{}{} }()
		case <-ct.done:
			return
		}
	}
}

func (ct *SimulatedContract) Start() {
	ct.start.Do(func() {
		go ct.run()
		go ct.forwardHeads(context.Background())
	})
}

func (ct *SimulatedContract) Stop() {
	ct.src.Unsubscribe(ct.subscription)
	close(ct.done)
}