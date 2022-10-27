package main

import (
	"io"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/simulators"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
)

func main() {
	f, err := os.OpenFile("output.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	rb := config.RunBook{
		Nodes:             4,
		AvgNetworkLatency: config.Duration(50 * time.Millisecond),
		RPCDetail: config.RPC{
			MaxBlockDelay:  2000,
			AverageLatency: 200,
		},
		BlockCadence: config.Blocks{
			Genesis:  big.NewInt(128_943_862),
			Cadence:  12 * time.Second,
			Duration: 50,
		},
		ConfigEvents: []config.ConfigEvent{
			{
				Block:          big.NewInt(128_943_863),
				F:              1,
				Offchain:       []byte(`{"targetProbability":"0.99","targetInRounds":6,"uniqueReports":false}`),
				Rmax:           uint8(7),
				DeltaProgress:  config.Duration(10 * time.Second),
				DeltaResend:    config.Duration(10 * time.Second),
				DeltaRound:     config.Duration(2 * time.Second),
				DeltaGrace:     config.Duration(500 * time.Millisecond),
				DeltaStage:     config.Duration(2 * time.Second),
				MaxObservation: config.Duration(time.Second),
				MaxReport:      config.Duration(500 * time.Millisecond),
				MaxAccept:      config.Duration(50 * time.Millisecond),
				MaxTransmit:    config.Duration(50 * time.Millisecond),
			},
		},
		Upkeeps: []config.Upkeep{
			{Count: 15, StartID: big.NewInt(200), GenerateFunc: "24x - 3", OffsetFunc: "3x + 4"},
			{Count: 30, StartID: big.NewInt(400), GenerateFunc: "12x - 1", OffsetFunc: "2x + 5"},
			{Count: 20, StartID: big.NewInt(600), GenerateFunc: "6x - 5", OffsetFunc: "2x + 1"},
		},
	}

	log.Println("generating upkeeps")
	upkeeps, err := simulators.GenerateSimulatedUpkeeps(rb)
	if err != nil {
		panic(err)
	}

	// generic report encoder for testing evm encoding/decoding
	enc := chain.NewEVMReportEncoder()

	// generic config digester
	digester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1,
		ContractAddress: common.BigToAddress(big.NewInt(12)),
	}

	ngConf := NodeGroupConfig{
		Digester:      digester,
		Cadence:       rb.BlockCadence,
		Encoder:       enc,
		Upkeeps:       upkeeps,
		ConfigEvents:  rb.ConfigEvents,
		MonitorIO:     io.Discard, // TODO: the monitor data format is not text. not sure what to make of it yet.
		RPCConfig:     rb.RPCDetail,
		AvgNetLatency: rb.AvgNetworkLatency.Value(),
		LogPath:       "./ocr_logs",
		Logger:        log.Default(),
	}
	ng := NewNodeGroup(ngConf)

	if err := ng.Start(rb.Nodes); err != nil {
		log.Printf("%s", err)
	}
}
