package plugin

import (
	"context"
	"errors"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
)

const (
	OutcomeHistoryLimit = 10
)

type AutomationReportInfo struct{}

type Encoder interface {
	Encode(...ocr2keepers.CheckResult) ([]byte, error)
	Extract([]byte) ([]ocr2keepers.ReportedUpkeep, error)
}

type Coordinator interface {
	Accept(ocr2keepers.ReportedUpkeep) error
	IsTransmissionConfirmed(ocr2keepers.ReportedUpkeep) bool
}

type ocr3Plugin struct {
	PrebuildHooks []func(ocr2keepersv3.AutomationOutcome) error
	BuildHooks    []func(*ocr2keepersv3.AutomationObservation) error
	ReportEncoder Encoder
	Coordinators  []Coordinator
	Services      []service.Recoverable
	Config        config.OffchainConfig
	Logger        *log.Logger
}

func (plugin *ocr3Plugin) Query(ctx context.Context, outctx ocr3types.OutcomeContext) (types.Query, error) {
	return nil, nil
}

func (plugin *ocr3Plugin) Observation(ctx context.Context, outcome ocr3types.OutcomeContext, query types.Query) (types.Observation, error) {
	// first round outcome will be nil or empty so no processing should be done
	if outcome.PreviousOutcome != nil || len(outcome.PreviousOutcome) != 0 {
		// Decode the outcome to AutomationOutcome
		automationOutcome, err := ocr2keepersv3.DecodeAutomationOutcome(outcome.PreviousOutcome)
		if err != nil {
			return nil, err
		}

		// TODO: validate outcome (even though it is a signed outcome)

		// Execute pre-build hooks
		plugin.Logger.Printf("running pre-build hooks")
		for _, hook := range plugin.PrebuildHooks {
			err = errors.Join(err, hook(automationOutcome))
		}
		if err != nil {
			return nil, err
		}
	}

	// Create new AutomationObservation
	observation := ocr2keepersv3.AutomationObservation{}

	// Execute build hooks
	plugin.Logger.Printf("running build hooks")
	for _, hook := range plugin.BuildHooks {
		err := hook(&observation)
		if err != nil {
			return nil, err
		}
	}

	// Encode the observation to bytes
	encoded, err := observation.Encode()
	if err != nil {
		return nil, err
	}

	// Return the encoded bytes as ocr3 observation
	return types.Observation(encoded), nil
}

func (plugin *ocr3Plugin) ValidateObservation(outctx ocr3types.OutcomeContext, query types.Query, ao types.AttributedObservation) error {
	return nil
}

func (plugin *ocr3Plugin) Outcome(outctx ocr3types.OutcomeContext, query types.Query, attributedObservations []types.AttributedObservation) (ocr3types.Outcome, error) {
	p := newPerformables(len(attributedObservations) / 2)
	c := newCoordinateBlock(len(attributedObservations) / 2)

	// extract observations and pass them on to evaluators
	for _, attributedObservation := range attributedObservations {
		observation, err := ocr2keepersv3.DecodeAutomationObservation(attributedObservation.Observation)
		if err != nil {
			return nil, err
		}

		// TODO: validate incoming observation

		p.add(observation)
		c.add(observation)
	}

	var latestIdx int
	var history []ocr2keepersv3.BasicOutcome

	if outctx.SeqNr != 1 {
		prev, err := ocr2keepersv3.DecodeAutomationOutcome(outctx.PreviousOutcome)
		if err != nil {
			return nil, err
		}

		// TODO: validate outcome (even though it is a signed outcome)

		latestIdx = prev.LatestIdx

		if len(prev.History) < OutcomeHistoryLimit {
			history = make([]ocr2keepersv3.BasicOutcome, len(prev.History))
			copy(history, prev.History[:len(prev.History)])
		} else {
			history = make([]ocr2keepersv3.BasicOutcome, OutcomeHistoryLimit)
			copy(history, prev.History[:OutcomeHistoryLimit])
		}
	}

	outcome := ocr2keepersv3.AutomationOutcome{
		BasicOutcome: ocr2keepersv3.BasicOutcome{
			Metadata: make(map[ocr2keepersv3.OutcomeMetadataKey]interface{}),
		},
	}

	p.set(&outcome)
	c.set(&outcome)

	// advance the latest by 1 and apply the basic outcome to the history
	latestIdx++
	if latestIdx >= OutcomeHistoryLimit {
		latestIdx = 0
	}

	if len(history) < OutcomeHistoryLimit {
		// if the history is still less than the limit the new value can be
		// safely appended
		history = append(history, outcome.BasicOutcome)
	} else {
		// if the history is at the limit the latest value reset the value at
		// the latest index. this creates a ring buffer of history values
		history[latestIdx] = outcome.BasicOutcome
	}

	outcome.History = history
	outcome.LatestIdx = latestIdx

	plugin.Logger.Printf("returning outcome with %d results", len(outcome.Performable))

	return outcome.Encode()
}

func (plugin *ocr3Plugin) Reports(_ uint64, raw ocr3types.Outcome) ([]ocr3types.ReportWithInfo[AutomationReportInfo], error) {
	var (
		reports []ocr3types.ReportWithInfo[AutomationReportInfo]
		outcome ocr2keepersv3.AutomationOutcome
		err     error
	)

	if outcome, err = ocr2keepersv3.DecodeAutomationOutcome(raw); err != nil {
		return nil, err
	}

	// TODO: validate outcome (even though it is a signed outcome)

	plugin.Logger.Printf("creating report from outcome with %d results", len(outcome.Performable))

	toPerform := []ocr2keepers.CheckResult{}
	var gasUsed uint64

	for i, result := range outcome.Performable {
		if gasUsed+result.GasAllocated+uint64(plugin.Config.GasOverheadPerUpkeep) > uint64(plugin.Config.GasLimitPerReport) || len(toPerform) > plugin.Config.MaxUpkeepBatchSize {
			// encode current collection
			encoded, encodeErr := plugin.ReportEncoder.Encode(toPerform...)
			err = errors.Join(err, encodeErr)

			if encodeErr == nil {
				// add encoded data to reports
				reports = append(reports, ocr3types.ReportWithInfo[AutomationReportInfo]{
					Report: types.Report(encoded),
				})

				// reset collection
				toPerform = []ocr2keepers.CheckResult{}
				gasUsed = 0
			}
		}

		gasUsed += result.GasAllocated + uint64(plugin.Config.GasOverheadPerUpkeep)
		toPerform = append(toPerform, outcome.Performable[i])
	}

	// if there are still values to add
	if len(toPerform) > 0 {
		// encode current collection
		encoded, encodeErr := plugin.ReportEncoder.Encode(toPerform...)
		err = errors.Join(err, encodeErr)

		if encodeErr == nil {
			// add encoded data to reports
			reports = append(reports, ocr3types.ReportWithInfo[AutomationReportInfo]{
				Report: types.Report(encoded),
				Info:   AutomationReportInfo{},
			})
		}
	}

	return reports, err
}

func (plugin *ocr3Plugin) ShouldAcceptFinalizedReport(_ context.Context, _ uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	for _, upkeep := range upkeeps {
		plugin.Logger.Printf("accepting upkeep by id '%s'", upkeep.UpkeepID)
		for _, coord := range plugin.Coordinators {
			if err := coord.Accept(upkeep); err != nil {
				plugin.Logger.Printf("failed to accept upkeep by id '%s', error is %v", upkeep.UpkeepID, err)
			}
		}
	}

	return true, nil
}

func (plugin *ocr3Plugin) ShouldTransmitAcceptedReport(_ context.Context, _ uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	for _, upkeep := range upkeeps {
		// if any upkeep in the report does not have confirmations from all coordinators, attempt again
		allConfirmationsFalse := true
		for _, coord := range plugin.Coordinators {
			if coord.IsTransmissionConfirmed(upkeep) {
				allConfirmationsFalse = false
			}
			plugin.Logger.Printf("checking transmit of upkeep '%s' %t", upkeep.UpkeepID, coord.IsTransmissionConfirmed(upkeep))
		}

		if allConfirmationsFalse {
			return true, nil
		}

	}

	return false, nil
}

func (plugin *ocr3Plugin) Close() error {
	var err error

	for i := range plugin.Services {
		err = errors.Join(err, plugin.Services[i].Close())
	}

	return err
}

// this start function should not block
func (plugin *ocr3Plugin) startServices() {
	for i := range plugin.Services {
		go func(svc service.Recoverable) {
			_ = svc.Start(context.Background())
		}(plugin.Services[i])
	}
}
