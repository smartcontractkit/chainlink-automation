package plugin

import (
	"context"

	"github.com/smartcontractkit/libocr/commontypes"
	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2plus"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
)

// DelegateConfig provides a single configuration struct for all options
// to be passed to the oracle, oracle factory, and underlying plugin/services.
type DelegateConfig struct {
	BinaryNetworkEndpointFactory types.BinaryNetworkEndpointFactory
	V2Bootstrappers              []commontypes.BootstrapperLocator
	ContractConfigTracker        types.ContractConfigTracker
	ContractTransmitter          types.ContractTransmitter
	KeepersDatabase              ocr3types.Database
	Logger                       commontypes.Logger
	MonitoringEndpoint           commontypes.MonitoringEndpoint
	OffchainConfigDigester       types.OffchainConfigDigester
	OffchainKeyring              types.OffchainKeyring
	OnchainKeyring               types.OnchainKeyring
	LocalConfig                  types.LocalConfig

	// LogProvider allows reads on the latest log events ready to be processed
	LogProvider flows.LogEventProvider

	// EventProvider allows reads on latest transmit events
	EventProvider coordinator.EventProvider

	// Encoder provides methods to encode/decode reports
	Encoder Encoder
}

// Delegate is a container struct for an Oracle plugin. This struct provides
// the ability to start and stop underlying services associated with the
// plugin instance.
type Delegate struct {
	keeper *offchainreporting.Oracle
}

// NewDelegate provides a new Delegate from a provided config. A new logger
// is defined that wraps the configured logger with a default Go logger.
// The plugin uses a *log.Logger by default so all log output from the
// built-in logger are written to the provided logger as Debug logs prefaced
// with '[keepers-plugin] ' and a short file name.
func NewDelegate(c DelegateConfig) (*Delegate, error) {

	return &Delegate{keeper: nil}, nil
}

// Start starts the OCR oracle and any associated services
func (d *Delegate) Start(_ context.Context) error {
	return nil
}

// Close stops the OCR oracle and any associated services
func (d *Delegate) Close() error {
	return nil
}
