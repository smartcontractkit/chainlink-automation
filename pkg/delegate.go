package ocr2keepers

import (
	"fmt"
	"log"

	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
)

type Delegate struct {
	keeper *offchainreporting.Oracle
}

func NewDelegate(c DelegateConfig) (*Delegate, error) {
	wrapper := &logWriter{l: c.Logger}
	l := log.New(wrapper, "[keepers-plugin] ", log.Lshortfile)

	keeper, err := offchainreporting.NewOracle(offchainreporting.OracleArgs{
		BinaryNetworkEndpointFactory: c.BinaryNetworkEndpointFactory,
		V2Bootstrappers:              c.V2Bootstrappers,
		ContractConfigTracker:        c.ContractConfigTracker,
		ContractTransmitter:          c.ContractTransmitter,
		Database:                     c.KeepersDatabase,
		LocalConfig:                  c.LocalConfig,
		Logger:                       c.Logger,
		MonitoringEndpoint:           c.MonitoringEndpoint,
		OffchainConfigDigester:       c.OffchainConfigDigester,
		OffchainKeyring:              c.OffchainKeyring,
		OnchainKeyring:               c.OnchainKeyring,
		ReportingPluginFactory:       keepers.NewReportingPluginFactory(c.Registry, c.ReportEncoder, l),
	})

	// TODO: handle errors better
	if err != nil {
		return nil, err
	}

	return &Delegate{keeper: keeper}, nil
}

func (d *Delegate) Start() error {
	if err := d.keeper.Start(); err != nil {
		return fmt.Errorf("%w: starting keeper oracle", err)
	}
	return nil
}

func (d *Delegate) Close() error {
	if err := d.keeper.Close(); err != nil {
		return fmt.Errorf("%w: stopping keeper oracle", err)
	}
	return nil
}
