package plugin

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"golang.org/x/crypto/sha3"

	"github.com/smartcontractkit/ocr2keepers/pkg/config"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	OutcomeHistoryLimit = 10
	OutcomeSamplesLimit = 100
)

type AutomationReportInfo struct{}

//go:generate mockery --name Coordinator --structname MockCoordinator --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin" --case underscore --filename coordinator.generated.go
type Coordinator interface {
	Accept(ocr2keepers.ReportedUpkeep) error
	IsTransmissionConfirmed(ocr2keepers.ReportedUpkeep) bool
}

type ocr3Plugin struct {
	ConfigDigest  types.ConfigDigest
	PrebuildHooks []func(ocr2keepersv3.AutomationOutcome) error
	BuildHooks    []func(*ocr2keepersv3.AutomationObservation) error
	ReportEncoder ocr2keepers.Encoder
	Coordinators  []Coordinator
	Services      []service.Recoverable
	Config        config.OffchainConfig
	F             int
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

		// validate outcome (even though it is a signed outcome)
		if err := ocr2keepersv3.ValidateAutomationOutcome(automationOutcome); err != nil {
			return nil, err
		}

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
	observation := ocr2keepersv3.AutomationObservation{
		Metadata: make(map[ocr2keepersv3.ObservationMetadataKey]interface{}),
	}

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
	o, err := ocr2keepersv3.DecodeAutomationObservation(ao.Observation)
	if err != nil {
		return err
	}

	return ocr2keepersv3.ValidateAutomationObservation(o)
}

func (plugin *ocr3Plugin) Outcome(outctx ocr3types.OutcomeContext, query types.Query, attributedObservations []types.AttributedObservation) (ocr3types.Outcome, error) {
	p := newPerformables(plugin.F + 1)
	c := newCoordinateBlock(len(attributedObservations) / 2)
	s := newSamples(OutcomeSamplesLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))
	r := newRecoverables(plugin.F + 1)

	// extract observations and pass them on to evaluators
	for _, attributedObservation := range attributedObservations {
		observation, err := ocr2keepersv3.DecodeAutomationObservation(attributedObservation.Observation)
		if err != nil {
			return nil, err
		}

		if err := ocr2keepersv3.ValidateAutomationObservation(observation); err != nil {
			plugin.Logger.Printf("invalid observation from oracle %d in sequence %d", attributedObservation.Observer, outctx.SeqNr)

			// if the validation for an observation fails at this point, discard
			// the observation and move to the next one
			continue
		}

		p.add(observation)
		c.add(observation)
		s.add(observation)
		r.add(observation)
	}

	outcome := ocr2keepersv3.AutomationOutcome{
		BasicOutcome: ocr2keepersv3.BasicOutcome{
			Metadata: make(map[ocr2keepersv3.OutcomeMetadataKey]interface{}),
		},
	}

	p.set(&outcome)
	c.set(&outcome)
	s.set(&outcome)
	r.set(&outcome)

	var previous *ocr2keepersv3.AutomationOutcome
	if outctx.SeqNr != 1 {
		prev, err := ocr2keepersv3.DecodeAutomationOutcome(outctx.PreviousOutcome)
		if err != nil {
			return nil, err
		}

		// validate outcome (even though it is a signed outcome)
		if err := ocr2keepersv3.ValidateAutomationOutcome(prev); err != nil {
			return nil, err
		}

		previous = &prev
	}

	// set the latest value in the history
	UpdateHistory(previous, &outcome, OutcomeHistoryLimit)

	plugin.Logger.Printf("returning outcome with %d results", len(outcome.Performable))

	return outcome.Encode()
}

func (plugin *ocr3Plugin) Reports(seqNr uint64, raw ocr3types.Outcome) ([]ocr3types.ReportWithInfo[AutomationReportInfo], error) {
	var (
		reports []ocr3types.ReportWithInfo[AutomationReportInfo]
		outcome ocr2keepersv3.AutomationOutcome
		err     error
	)

	if plugin.Config.MaxUpkeepBatchSize <= 0 {
		return nil, fmt.Errorf("invalid max upkeep batch size: %d", plugin.Config.MaxUpkeepBatchSize)
	}

	if plugin.Config.GasLimitPerReport == 0 {
		return nil, fmt.Errorf("invalid gas limit per report: %d", plugin.Config.GasLimitPerReport)
	}

	if outcome, err = ocr2keepersv3.DecodeAutomationOutcome(raw); err != nil {
		return nil, err
	}

	// validate outcome (even though it is a signed outcome)
	if err := ocr2keepersv3.ValidateAutomationOutcome(outcome); err != nil {
		return nil, err
	}

	plugin.Logger.Printf("creating report from outcome with %d results; max batch size: %d; report gas limit %d", len(outcome.Performable), plugin.Config.MaxUpkeepBatchSize, plugin.Config.GasLimitPerReport)

	toPerform := []ocr2keepers.CheckResult{}
	var gasUsed uint64

	for i, result := range outcome.Performable {
		if len(toPerform) >= plugin.Config.MaxUpkeepBatchSize || gasUsed+result.GasAllocated+uint64(plugin.Config.GasOverheadPerUpkeep) > uint64(plugin.Config.GasLimitPerReport) {
			if len(toPerform) > 0 {
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
		}

		gasUsed += result.GasAllocated + uint64(plugin.Config.GasOverheadPerUpkeep)
		toPerform = append(toPerform, outcome.Performable[i])
	}

	// if there are still values to add
	if len(toPerform) > 0 && gasUsed <= uint64(plugin.Config.GasLimitPerReport) {
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

	plugin.Logger.Printf("%d reports created for sequence number %d", len(reports), seqNr)

	if err != nil {
		plugin.Logger.Printf("error encountered while encoding: %s", err)
	}

	return reports, nil
}

func (plugin *ocr3Plugin) ShouldAcceptAttestedReport(_ context.Context, seqNr uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	plugin.Logger.Printf("inside should accept attested report for sequence number %d", seqNr)
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	plugin.Logger.Printf("%d upkeeps found in report for should accept attested for sequence number %d", len(upkeeps), seqNr)

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

func (plugin *ocr3Plugin) ShouldTransmitAcceptedReport(_ context.Context, seqNr uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	plugin.Logger.Printf("%d upkeeps found in report for should transmit for sequence number %d", len(upkeeps), seqNr)

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
			if err := svc.Start(context.Background()); err != nil {
				plugin.Logger.Printf("error starting plugin services: %s", err)
			}
		}(plugin.Services[i])
	}
}

// Generates a randomness source derived from the config and seq # so
// that it's the same across the network for the same round
func getRandomKeySource(cd types.ConfigDigest, seqNr uint64) [16]byte {
	// similar key building as libocr transmit selector
	hash := sha3.NewLegacyKeccak256()
	hash.Write(cd[:])
	temp := make([]byte, 8)
	binary.LittleEndian.PutUint64(temp, seqNr)
	hash.Write(temp)

	var keyRandSource [16]byte
	copy(keyRandSource[:], hash.Sum(nil))
	return keyRandSource
}
