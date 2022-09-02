package ocr2keepers

import (
	"fmt"
	"time"

	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
)

type Delegate struct {
	keeper *offchainreporting.Oracle
}

func NewDelegate(c DelegateConfig) (*Delegate, error) {
	keeper, err := offchainreporting.NewOracle(offchainreporting.OracleArgs{
		BinaryNetworkEndpointFactory: c.BinaryNetworkEndpointFactory,
		V2Bootstrappers:              c.V2Bootstrappers,
		ContractConfigTracker:        c.ContractConfigTracker,
		ContractTransmitter:          c.ContractTransmitter,
		Database:                     c.KeepersDatabase,
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  1 * time.Second,        // TODO: choose sane configs
			ContractConfigTrackerPollInterval:  15 * time.Second,       // TODO: choose sane configs
			ContractTransmitterTransmitTimeout: 1 * time.Second,        // TODO: choose sane configs
			DatabaseTimeout:                    100 * time.Millisecond, // TODO: choose sane configs
			ContractConfigConfirmations:        1,                      // TODO: choose sane configs
		},
		Logger:                 c.Logger,
		MonitoringEndpoint:     c.MonitoringEndpoint,
		OffchainConfigDigester: c.OffchainConfigDigester,
		OffchainKeyring:        c.OffchainKeyring,
		OnchainKeyring:         c.OnchainKeyring,
		ReportingPluginFactory: keepers.NewReportingPluginFactory(c.Registry),
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
