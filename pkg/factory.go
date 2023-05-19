package ocr2keepers

import (
	"fmt"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
)

// an upkeep key is composed of a block number and upkeep id (~40 bytes)
// an observation is multiple upkeeps to be performed
// 100 upkeeps to be performed would be a very high upper limit
// 100 * 10 = 1_000 bytes
const MaxObservationLength = 1_000

// a report is composed of 1 or more abi encoded perform calls
// with performData of arbitrary length
// TODO (config): pick sane limit based on expected performData size. Maybe set
// this to block size limit or 2/3 block size limit?
// TODO (config): Also might need to be part of the off-chain config instead of
// a constant.
const MaxReportLength = 10_000

type CoordinatorFactory interface {
	NewCoordinator(config.OffchainConfig) (Coordinator, error)
}

type ConditionalObserverFactory interface {
	NewConditionalObserver(config.OffchainConfig, types.ReportingPluginConfig, Coordinator) (ConditionalObserver, error)
}

func NewReportingPluginFactory(
	encoder Encoder, // Encoder should be a static implementation with no state
	executer Executer,
	coordinatorFactory CoordinatorFactory,
	condObserverFactory ConditionalObserverFactory,
	logger *log.Logger,
) types.ReportingPluginFactory {
	factory := &pluginFactory{
		encoder:             encoder,
		executer:            executer,
		coordinatorFactory:  coordinatorFactory,
		condObserverFactory: condObserverFactory,
		logger:              logger,
	}

	return factory
}

type PluginStarterCloser interface {
	Start()
	Close() error
}

type pluginFactory struct {
	encoder             Encoder
	executer            Executer
	coordinatorFactory  CoordinatorFactory
	condObserverFactory ConditionalObserverFactory
	logger              *log.Logger
}

func (f *pluginFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	offChainCfg, err := config.DecodeOffchainConfig(c.OffchainConfig)
	if err != nil {
		return nil, types.ReportingPluginInfo{}, fmt.Errorf("%w: failed to decode off chain config", err)
	}

	info := types.ReportingPluginInfo{
		Name: fmt.Sprintf("Oracle %d: Keepers Plugin Instance w/ Digest '%s'", c.OracleID, c.ConfigDigest),
		Limits: types.ReportingPluginLimits{
			// queries should be empty with the current implementation
			MaxQueryLength:       0,
			MaxObservationLength: MaxObservationLength,
			MaxReportLength:      MaxReportLength,
		},
		// UniqueReports increases the threshold of signatures needed for quorum
		// to (n+f)/2 so that it's guaranteed a unique report is generated per
		// round. Fixed to false for ocr2keepers, as we always expect f+1
		// signatures on a report on contract and do not support uniqueReports
		// quorum.
		UniqueReports: false,
	}

	coordinator, err := f.coordinatorFactory.NewCoordinator(offChainCfg)
	if err != nil {
		return nil, info, err
	}

	condObserver, err := f.condObserverFactory.NewConditionalObserver(offChainCfg, c, coordinator)
	if err != nil {
		return nil, info, err
	}

	// for each of the provided dependencies, check if they satisfy a start/stop
	// interface. if so, add them to a services array so that the plugin can
	// shut them down.
	possibleSrvs := []interface{}{coordinator, condObserver, f.executer}
	subProcs := make([]PluginStarterCloser, 0, len(possibleSrvs))
	for _, possibleSrv := range possibleSrvs {
		if sub, ok := possibleSrv.(PluginStarterCloser); ok {
			sub.Start()
			subProcs = append(subProcs, sub)
		}
	}

	return &ocrPlugin{
		encoder:        f.encoder,
		executer:       f.executer,
		coordinator:    coordinator, // coordinator is a service that should have a start / stop method
		condObserver:   condObserver,
		logger:         f.logger,
		subProcs:       subProcs,
		conf:           offChainCfg,
		mercuryEnabled: false,
	}, info, nil
}
