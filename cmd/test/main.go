package main

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/blocks"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/simulators"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func main() {

	rb := config.RunBook{
		BlockCadence: config.Blocks{
			Genesis:  big.NewInt(128_943_862),
			Cadence:  14 * time.Second,
			Duration: 10,
		},
		Configs: map[uint64]config.Config{
			1: {
				Count: 1,
				Signers: [][]byte{
					[]byte(``),
				},
				Transmitters: []string{
					"",
				},
				F:               1,
				Onchain:         []byte(`{}`),
				OffchainVersion: 1,
				Offchain:        []byte(`{}`),
			},
		},
		Upkeeps: []config.Upkeep{
			{Count: 15, StartID: big.NewInt(200), GenerateFunc: "24x - 3", OffsetFunc: "3x + 4"},
		},
	}
	upkeeps, err := simulators.GenerateSimulatedUpkeeps(rb)
	if err != nil {
		panic(err)
	}

	enc := chain.NewEVMReportEncoder()
	blocks := blocks.NewBlockBroadcaster(rb.BlockCadence)
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1,
		ContractAddress: common.BigToAddress(big.NewInt(12)),
	}

	ct := simulators.NewSimulatedContract(blocks, digester, upkeeps, enc, log.Default())
	slogger := NewSimpleLogger(log.Writer(), Debug)
	simNet := simulators.NewSimulatedNetwork()
	db := simulators.NewSimulatedDatabase()
	monitor := NewMonitorToWriter(log.Writer())

	offKeyRing, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)
	if err != nil {
		panic(err)
	}

	onKeyRing, err := config.NewEVMKeyring(rand.Reader)
	if err != nil {
		panic(err)
	}

	config := ocr2keepers.DelegateConfig{
		BinaryNetworkEndpointFactory: simNet.NewFactory(),
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		ContractConfigTracker:        ct,
		ContractTransmitter:          ct,
		KeepersDatabase:              db,
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  time.Second,
			ContractConfigConfirmations:        1,
			SkipContractConfigConfirmations:    false,
			ContractConfigTrackerPollInterval:  time.Second,
			ContractTransmitterTransmitTimeout: time.Second,
			DatabaseTimeout:                    time.Second,
			DevelopmentMode:                    "",
		},
		Logger:                 slogger,
		MonitoringEndpoint:     monitor,
		OffchainConfigDigester: digester,
		OffchainKeyring:        offKeyRing,
		OnchainKeyring:         onKeyRing,
		Registry:               ct,
		PerformLogProvider:     ct,
		ReportEncoder:          enc,
	}

	service, err := ocr2keepers.NewDelegate(config)
	if err != nil {
		panic(err)
	}
	_ = service.Start(context.Background())
}
