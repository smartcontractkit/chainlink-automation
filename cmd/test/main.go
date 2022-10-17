package main

import (
	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting/types"
	protocol "github.com/smartcontractkit/libocr/offchainreporting2"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/mocks"
)

func main() {

	var d types.Database
	args := protocol.OracleArgs{
		BinaryNetworkEndpointFactory: new(mocks.MockBinaryNetworkEndpointFactory),
		V2Bootstrappers:              []commontypes.BootstrapperLocator{},
		ContractConfigTracker:        new(mocks.MockContractConfigTracker),
		ContractTransmitter:          new(mocks.MockContractTransmitter),
	}
	_, err := protocol.NewOracle(args)
	if err != nil {
		panic(err)
	}

}
