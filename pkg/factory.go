package ocr2keepers

import (
	"fmt"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
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
	NewCoordinator() (Coordinator, error)
}

type ConditionalObserverFactory interface {
	NewConditionalObserver() (ConditionalObserver, error)
}

func NewReportingPluginFactory(
	encoder Encoder, // Encoder should be a static implementation with no state
	coordinatorFactory CoordinatorFactory,
	condObserverFactory ConditionalObserverFactory,
	logger *log.Logger,
) types.ReportingPluginFactory {
	return &pluginFactory{}
}

type PluginCloser interface {
	Close() error
}

type pluginFactory struct {
	encoder             Encoder
	coordinatorFactory  CoordinatorFactory
	condObserverFactory ConditionalObserverFactory
	logger              *log.Logger
}

func (f *pluginFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	// TODO: decode off-chain config

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

	// TODO: need to pass the off-chain config to the coordinator factory
	coordinator, err := f.coordinatorFactory.NewCoordinator()
	if err != nil {
		return nil, info, err
	}

	// TODO: need to pass the off-chain config to the observer factory
	condObserver, err := f.condObserverFactory.NewConditionalObserver()
	if err != nil {
		return nil, info, err
	}

	// for each of the provided dependencies, check if they satisfy a start/stop
	// interface. if so, add them to a services array so that the plugin can
	// shut them down.
	possibleSrvs := []interface{}{coordinator, condObserver}
	subProcs := make([]PluginCloser, 0, len(possibleSrvs))
	for x := 0; x < len(possibleSrvs); x++ {
		sub, ok := possibleSrvs[x].(PluginCloser)
		if ok {
			subProcs = append(subProcs, sub)
		}
	}

	// TODO: provide off-chain config to plugin

	return &ocrPlugin{
		encoder:            f.encoder,
		coordinator:        coordinator, // coordinator is a service that should have a start / stop method
		condObserver:       condObserver,
		logger:             f.logger,
		subProcs:           subProcs,
		upkeepGasOverhead:  0, // TODO: needs to come from config
		reportGasLimit:     0, // TODO: needs to come from config
		maxUpkeepBatchSize: 0, // TODO: needs to come from config
	}, info, nil
}
