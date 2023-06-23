package plugin

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2plus"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/automationshim"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
)

var (
	newOracleFn = offchainreporting.NewOracle
)

type oracle interface {
	Start() error
	Close() error
}

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

	// Runnable is a check pipeline runner
	Runnable runner.Runnable

	// Encoder provides methods to encode/decode reports
	Encoder Encoder

	// CacheExpiration is the duration of time a cached key is available. Use
	// this value to balance memory usage and RPC calls. A new set of keys is
	// generated with every block so a good setting might come from block time
	// times number of blocks of history to support not replaying reports.
	CacheExpiration time.Duration

	// CacheEvictionInterval is a parameter for how often the cache attempts to
	// evict expired keys. This value should be short enough to ensure key
	// eviction doesn't block for too long, and long enough that it doesn't
	// cause frequent blocking.
	CacheEvictionInterval time.Duration

	// MaxServiceWorkers is the total number of go-routines allowed to make RPC
	// simultaneous calls on behalf of the sampling operation. This parameter
	// is 10x the number of available CPUs by default. The RPC calls are memory
	// heavy as opposed to CPU heavy as most of the work involves waiting on
	// network responses.
	MaxServiceWorkers int

	// ServiceQueueLength is the buffer size for the RPC service queue. Fewer
	// workers or slower RPC responses will cause this queue to build up.
	// Adding new items to the queue will block if the queue becomes full.
	ServiceQueueLength int
}

// Delegate is a container struct for an Oracle plugin. This struct provides
// the ability to start and stop underlying services associated with the
// plugin instance.
type Delegate struct {
	keeper oracle
}

// NewDelegate provides a new Delegate from a provided config. A new logger
// is defined that wraps the configured logger with a default Go logger.
// The plugin uses a *log.Logger by default so all log output from the
// built-in logger are written to the provided logger as Debug logs prefaced
// with '[keepers-plugin] ' and a short file name.
func NewDelegate(c DelegateConfig) (*Delegate, error) {
	// set some defaults
	conf := config.ReportingFactoryConfig{
		CacheExpiration:       config.DefaultCacheExpiration,
		CacheEvictionInterval: config.DefaultCacheClearInterval,
		MaxServiceWorkers:     config.DefaultMaxServiceWorkers,
		ServiceQueueLength:    config.DefaultServiceQueueLength,
	}

	// override if set in config
	if c.CacheExpiration != 0 {
		conf.CacheExpiration = c.CacheExpiration
	}

	if c.CacheEvictionInterval != 0 {
		conf.CacheEvictionInterval = c.CacheEvictionInterval
	}

	if c.MaxServiceWorkers != 0 {
		conf.MaxServiceWorkers = c.MaxServiceWorkers
	}

	if c.ServiceQueueLength != 0 {
		conf.ServiceQueueLength = c.ServiceQueueLength
	}

	// the log wrapper is to be able to use a log.Logger everywhere instead of
	// a variety of logger types. all logs write to the Debug method.
	wrapper := &logWriter{l: c.Logger}
	l := log.New(wrapper, "[keepers-plugin] ", log.Lshortfile)

	l.Printf("creating oracle with reporting factory config: %+v", conf)

	// create the oracle from config values
	keeper, err := newOracleFn(offchainreporting.AutomationOracleArgs{
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
		ReportingPluginFactory: NewReportingPluginFactory[automationshim.AutomationReportInfo](
			c.LogProvider,
			c.EventProvider,
			c.Runnable,
			runner.RunnerConfig{
				Workers:           conf.MaxServiceWorkers,
				WorkerQueueLength: conf.ServiceQueueLength,
				CacheExpire:       conf.CacheExpiration,
				CacheClean:        conf.CacheEvictionInterval,
			},
			c.Encoder,
			l,
		),
	})

	if err != nil {
		return nil, fmt.Errorf("%w: failed to create new OCR oracle", err)
	}

	return &Delegate{keeper: keeper}, nil
}

// Start starts the OCR oracle and any associated services
func (d *Delegate) Start(_ context.Context) error {
	if err := d.keeper.Start(); err != nil {
		return fmt.Errorf("%w: failed to start keeper oracle", err)
	}

	return nil
}

// Close stops the OCR oracle and any associated services
func (d *Delegate) Close() error {
	if err := d.keeper.Close(); err != nil {
		return fmt.Errorf("%w: failed to close keeper oracle", err)
	}

	return nil
}

type logWriter struct {
	l commontypes.Logger
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	l.l.Debug(string(p), nil)
	n = len(p)
	return
}
