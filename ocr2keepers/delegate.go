package ocr2keepers

import (
	"errors"
	"fmt"
	"time"

	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/keepers"
)

type Delegate struct {
	keeper *offchainreporting.Oracle
}

func NewDelegate(_ DelegateConfig) (*Delegate, error) {
	// TODO: handle errors here
	// TODO: define proper OracleArgs
	keeper, err := offchainreporting.NewOracle(offchainreporting.OracleArgs{
		//BinaryNetworkEndpointFactory: a.BinaryNetworkEndpointFactory,
		//V2Bootstrappers:              a.V2Bootstrappers,
		//ContractConfigTracker:        a.DKGContractConfigTracker,
		//ContractTransmitter:          a.DKGContractTransmitter,
		//Database:                     a.DKGDatabase,
		LocalConfig: types.LocalConfig{
			BlockchainTimeout:                  1 * time.Second,        // TODO: choose sane configs
			ContractConfigTrackerPollInterval:  15 * time.Second,       // TODO: choose sane configs
			ContractTransmitterTransmitTimeout: 1 * time.Second,        // TODO: choose sane configs
			DatabaseTimeout:                    100 * time.Millisecond, // TODO: choose sane configs
			ContractConfigConfirmations:        1,                      // TODO: choose sane configs
		},
		//Logger:                       a.DKGLogger,
		//MonitoringEndpoint:           a.DKGMonitoringEndpoint,
		//OffchainConfigDigester:       a.DKGOffchainConfigDigester,
		//OffchainKeyring:              a.OffchainKeyring,
		//OnchainKeyring:               a.OnchainKeyring,
		ReportingPluginFactory: keepers.NewReportingPluginFactory(),
	})

	if err != nil {
		return nil, err
	}

	return &Delegate{keeper: keeper}, nil
}

func (d *Delegate) Start() error {
	/*
		TODO: starting oracle throws an error without complete config
		if err := d.keeper.Start(); err != nil {
			return fmt.Errorf("%w: starting keeper oracle", err)
		}
	*/
	return errors.New("unimplemented")
}

func (d *Delegate) Close() error {
	if err := d.keeper.Close(); err != nil {
		return fmt.Errorf("%w: stopping keeper oracle", err)
	}
	return nil
}
