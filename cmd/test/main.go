package main

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/blocks"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/mocks"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/simulators"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func main() {

	rb := config.RunBook{
		BlockCadence: config.Blocks{
			Genesis: big.NewInt(128_943_862),
			Cadence: 14 * time.Second,
		},
	}

	blocks := blocks.NewBlockBroadcaster(rb.BlockCadence)
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1,
		ContractAddress: common.BigToAddress(big.NewInt(12)),
	}

	ct := simulators.NewSimulatedContract(blocks, digester)

	slogger := NewSimpleLogger(log.Writer(), Debug)
	simNet := simulators.NewSimulatedNetwork()

	config := ocr2keepers.DelegateConfig{
		BinaryNetworkEndpointFactory: simNet.NewFactory(),
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		ContractConfigTracker:        ct,
		ContractTransmitter:          new(mocks.MockContractTransmitter),
		KeepersDatabase:              new(mocks.MockDatabase),
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
		MonitoringEndpoint:     new(mocks.MockMonitoringEndpoint),
		OffchainConfigDigester: digester,
		OffchainKeyring:        new(mocks.MockOffchainKeyring),
		OnchainKeyring:         new(mocks.MockOnchainKeyring),
		Registry:               new(mocks.MockRegistry),
		PerformLogProvider:     new(mocks.MockPerformLogProvider),
		ReportEncoder:          chain.NewEVMReportEncoder(),
	}

	service, err := ocr2keepers.NewDelegate(config)
	if err != nil {
		panic(err)
	}
	service.Start(context.TODO())
}
