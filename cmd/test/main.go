package main

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/mocks"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/ocr2keepers/cmd/test/simulators"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func main() {

	var pData []byte
	var err error

	if pData, err = ioutil.ReadFile("./runbook.yaml"); err != nil {
		panic(err)
	}

	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(pData, &data)
	if err != nil {
		panic(err)
	}

	slogger := NewSimpleLogger(log.Writer(), Debug)
	simNet := simulators.NewSimulatedNetwork()

	config := ocr2keepers.DelegateConfig{
		BinaryNetworkEndpointFactory: simNet.NewFactory(),
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
