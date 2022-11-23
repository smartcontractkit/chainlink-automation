package keepers

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

const maxObservationLength = 1_000

var _ types.ReportingPluginFactory = (*keepersReportingFactory)(nil)

type ReportingFactoryConfig struct {
	CacheExpiration       time.Duration
	CacheEvictionInterval time.Duration
	MaxServiceWorkers     int
	ServiceQueueLength    int
}

type keepersReportingFactory struct {
	headSubscriber ktypes.HeadSubscriber
	registry       ktypes.Registry
	encoder        ktypes.ReportEncoder
	perfLogs       ktypes.PerformLogProvider
	logger         *log.Logger
	config         ReportingFactoryConfig
}

// NewReportingPluginFactory returns an OCR ReportingPluginFactory. When the plugin
// starts, a separate service is started as a separate go-routine automatically. There
// is no start or stop function for this service so stopping this service relies on
// releasing references to the plugin such that the Go garbage collector cleans up
// hanging routines automatically.
func NewReportingPluginFactory(
	headSubscriber ktypes.HeadSubscriber,
	registry ktypes.Registry,
	perfLogs ktypes.PerformLogProvider,
	encoder ktypes.ReportEncoder,
	logger *log.Logger,
	config ReportingFactoryConfig,
) types.ReportingPluginFactory {
	return &keepersReportingFactory{
		headSubscriber: headSubscriber,
		registry:       registry,
		perfLogs:       perfLogs,
		encoder:        encoder,
		logger:         logger,
		config:         config,
	}
}

// NewReportingPlugin implements the libocr/offchainreporting2/types ReportingPluginFactory interface
func (d *keepersReportingFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	offChainCfg, err := ktypes.DecodeOffchainConfig(c.OffchainConfig)
	if err != nil {
		return nil, types.ReportingPluginInfo{}, fmt.Errorf("%w: failed to decode off chain config", err)
	}

	info := types.ReportingPluginInfo{
		Name: fmt.Sprintf("Oracle %d: Keepers Plugin Instance w/ Digest '%s'", c.OracleID, c.ConfigDigest),
		Limits: types.ReportingPluginLimits{
			// queries should be empty anyway with the current implementation
			MaxQueryLength: 0,
			// an upkeep key is composed of a block number and upkeep id (~40 bytes)
			// an observation is multiple upkeeps to be performed
			// 100 upkeeps to be performed would be a very high upper limit
			// 100 * 10 = 1_000 bytes
			MaxObservationLength: maxObservationLength,
			// a report is composed of 1 or more abi encoded perform calls
			// with performData of arbitrary length
			MaxReportLength: 10_000, // TODO (config): pick sane limit based on expected performData size. maybe set this to block size limit or 2/3 block size limit?
		},
		UniqueReports: offChainCfg.UniqueReports,
	}

	// TODO (config): sample ratio is calculated with number of rounds, number
	// of nodes, and target probability for all upkeeps to be checked. each
	// chain will have a different average number of rounds per block. this
	// number needs to either come from a config, or be calculated on actual
	// performance of the nodes in real time. that is, start at 1 and increment
	// after some blocks pass until a stable number is reached.
	var p float64
	if len(offChainCfg.TargetProbability) == 0 {
		// TODO: Combine all default values in DecodeOffchainConfig
		offChainCfg.TargetProbability = "0.99999"
	}

	p, err = strconv.ParseFloat(offChainCfg.TargetProbability, 32)
	if err != nil {
		return nil, info, fmt.Errorf("%w: failed to parse configured probability", err)
	}

	if offChainCfg.TargetInRounds <= 0 {
		offChainCfg.TargetInRounds = 1
	}

	sample, err := sampleFromProbability(offChainCfg.TargetInRounds, c.N-c.F, float32(p))
	if err != nil {
		return nil, info, fmt.Errorf("%w: failed to create plugin", err)
	}

	service := newOnDemandUpkeepService(
		sample,
		d.headSubscriber,
		d.registry,
		d.logger,
		d.config.CacheExpiration,
		d.config.CacheEvictionInterval,
		d.config.MaxServiceWorkers,
		d.config.ServiceQueueLength)

	return &keepers{
		id:      c.OracleID,
		service: service,
		encoder: d.encoder,
		logger:  d.logger,
		filter:  newReportCoordinator(d.registry, time.Duration(offChainCfg.PerformLockoutWindow)*time.Millisecond, d.config.CacheEvictionInterval, d.perfLogs, d.logger),
	}, info, nil
}
