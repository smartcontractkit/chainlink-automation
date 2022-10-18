package main

import (
	"context"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/mocks"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func main() {

	slogger := NewSimpleLogger(log.Writer(), Debug)
	config := ocr2keepers.DelegateConfig{
		BinaryNetworkEndpointFactory: new(mocks.MockBinaryNetworkEndpointFactory),
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		ContractConfigTracker:        new(mocks.MockContractConfigTracker),
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
		OffchainConfigDigester: new(mocks.MockOffchainConfigDigester),
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
