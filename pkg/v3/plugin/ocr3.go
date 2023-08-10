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

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type AutomationReportInfo struct{}

//go:generate mockery --name Coordinator --structname MockCoordinator --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin" --case underscore --filename coordinator.generated.go
type Coordinator interface {
	ShouldAccept(ocr2keepers.ReportedUpkeep) bool
	ShouldTransmit(ocr2keepers.ReportedUpkeep) bool
	FilterResults([]ocr2keepers.CheckResult) ([]ocr2keepers.CheckResult, error)
	FilterProposals([]ocr2keepers.CoordinatedProposal) ([]ocr2keepers.CoordinatedProposal, error)
}

type ocr3Plugin struct {
	ConfigDigest                types.ConfigDigest
	ReportEncoder               ocr2keepers.Encoder
	Coordinator                 Coordinator
	RemoveFromStagingHook       RemoveFromStagingHook
	AddBlockHistoryHook         AddBlockHistoryHook
	AddFromStagingHook          AddFromStagingHook
	AddFromSamplesHook          AddFromSamplesHook
	AddLogRecoveryProposalsHook AddLogRecoveryProposalsHook
	Services                    []service.Recoverable
	Config                      config.OffchainConfig
	F                           int
	Logger                      *log.Logger
}

const (
	ObservationPerformablesLimit          = 100
	ObservationLogRecoveryProposalsLimit  = 2
	ObservationConditionalsProposalsLimit = 2
	ObservationBlockHistoryLimit          = 256
)

const (
	OutcomeAgreedPerformablesLimit = 100
	OutcomeAgreedProposalsLimit    = 50
	// TODO: Derive this limit from max checkPipelineTime and deltaRound
	OutcomeAgreedProposalsRoundHistoryLimit = 10
)

func (plugin *ocr3Plugin) Query(ctx context.Context, outctx ocr3types.OutcomeContext) (types.Query, error) {
	return nil, nil
}

func (plugin *ocr3Plugin) Observation(ctx context.Context, outctx ocr3types.OutcomeContext, query types.Query) (types.Observation, error) {
	// first round outcome will be nil or empty so no processing should be done
	if outctx.PreviousOutcome != nil || len(outctx.PreviousOutcome) != 0 {
		// Decode the outcome to AutomationOutcome
		automationOutcome, err := ocr2keepersv3.DecodeAutomationOutcome(outctx.PreviousOutcome)
		if err != nil {
			return nil, err
		}

		// validate outcome (even though it is a signed outcome)
		if err := ocr2keepersv3.ValidateAutomationOutcome(automationOutcome); err != nil {
			return nil, err
		}

		// Execute pre-build hooks
		plugin.RemoveFromStagingHook.RunHook(automationOutcome)
		// TODO Remove agreedSamples and logRecoveryProposals from metadata store
		// TODO Add coordinateFlow.ProcessOutcome
	}
	// Create new AutomationObservation
	observation := ocr2keepersv3.AutomationObservation{}
	plugin.AddBlockHistoryHook.RunHook(&observation, ObservationBlockHistoryLimit)
	plugin.AddFromStagingHook.RunHook(&observation, ObservationPerformablesLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))
	plugin.AddLogRecoveryProposalsHook.RunHook(&observation, ObservationLogRecoveryProposalsLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))
	plugin.AddFromSamplesHook.RunHook(&observation, ObservationConditionalsProposalsLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))

	plugin.Logger.Printf("built an observation in sequence nr %d with %d performables, %d upkeep proposals and %d block history", outctx.SeqNr, len(observation.Performable), len(observation.UpkeepProposals), len(observation.BlockHistory))

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
	p := newPerformables(plugin.F+1, OutcomeAgreedPerformablesLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))
	c := newCoordinatedProposals(plugin.F+1, OutcomeAgreedProposalsRoundHistoryLimit, OutcomeAgreedProposalsLimit, getRandomKeySource(plugin.ConfigDigest, outctx.SeqNr))

	// extract observations and pass them on to evaluators
	for _, attributedObservation := range attributedObservations {
		observation, err := ocr2keepersv3.DecodeAutomationObservation(attributedObservation.Observation)
		if err != nil {
			plugin.Logger.Printf("invalid observation from oracle %d in sequence %d", attributedObservation.Observer, outctx.SeqNr)

			// Ignore this observation and continue with further observations. It is expected we will get
			// atleast f+1 valid observations
			continue
		}

		if err := ocr2keepersv3.ValidateAutomationObservation(observation); err != nil {
			plugin.Logger.Printf("invalid observation from oracle %d in sequence %d", attributedObservation.Observer, outctx.SeqNr)

			// Ignore this observation and continue with further observations. It is expected we will get
			// atleast f+1 valid observations
			continue
		}

		plugin.Logger.Printf("adding observation from oracle %d in sequence %d with %d performables, %d upkeep proposals and %d block history",
			attributedObservation.Observer, outctx.SeqNr, len(observation.Performable), len(observation.UpkeepProposals), len(observation.BlockHistory))

		p.add(observation)
		c.add(observation)
	}

	outcome := ocr2keepersv3.AutomationOutcome{}
	prevOutcome := ocr2keepersv3.AutomationOutcome{}
	if outctx.PreviousOutcome != nil || len(outctx.PreviousOutcome) != 0 {
		// Decode the outcome to AutomationOutcome
		ao, err := ocr2keepersv3.DecodeAutomationOutcome(outctx.PreviousOutcome)
		if err != nil {
			return nil, err
		}
		// validate outcome (even though it is a signed outcome)
		if err := ocr2keepersv3.ValidateAutomationOutcome(ao); err != nil {
			return nil, err
		}
		prevOutcome = ao
	}

	p.set(&outcome)
	// Important to maintain the order here. Performables should be set before creating new proposals
	c.set(&outcome, prevOutcome)

	plugin.Logger.Printf("returning outcome with %d performables", len(outcome.AgreedPerformables))

	return outcome.Encode()
}

func (plugin *ocr3Plugin) Reports(seqNr uint64, raw ocr3types.Outcome) ([]ocr3types.ReportWithInfo[AutomationReportInfo], error) {
	var (
		reports []ocr3types.ReportWithInfo[AutomationReportInfo]
		outcome ocr2keepersv3.AutomationOutcome
		err     error
	)

	if outcome, err = ocr2keepersv3.DecodeAutomationOutcome(raw); err != nil {
		return nil, err
	}

	// validate outcome (even though it is a signed outcome)
	if err := ocr2keepersv3.ValidateAutomationOutcome(outcome); err != nil {
		return nil, err
	}

	plugin.Logger.Printf("creating report from outcome with %d agreed performables; max batch size: %d; report gas limit %d", len(outcome.AgreedPerformables), plugin.Config.MaxUpkeepBatchSize, plugin.Config.GasLimitPerReport)

	toPerform := []ocr2keepers.CheckResult{}
	var gasUsed uint64

	for i, result := range outcome.AgreedPerformables {
		if len(toPerform) >= plugin.Config.MaxUpkeepBatchSize ||
			gasUsed+result.GasAllocated+uint64(plugin.Config.GasOverheadPerUpkeep) > uint64(plugin.Config.GasLimitPerReport) {
			// If report has reached capacity, encode and append this report
			report, err := plugin.getReportFromPerformables(toPerform)
			if err != nil {
				return reports, fmt.Errorf("error encountered while encoding: %w", err)
			}
			// append to reports and reset collection
			reports = append(reports, report)
			toPerform = []ocr2keepers.CheckResult{}
			gasUsed = 0
		}

		// Add the result to current report
		gasUsed += result.GasAllocated + uint64(plugin.Config.GasOverheadPerUpkeep)
		toPerform = append(toPerform, outcome.AgreedPerformables[i])
	}

	// if there are still values to add
	if len(toPerform) > 0 {
		report, err := plugin.getReportFromPerformables(toPerform)
		if err != nil {
			return reports, fmt.Errorf("error encountered while encoding: %w", err)
		}
		reports = append(reports, report)
	}

	plugin.Logger.Printf("%d reports created for sequence number %d", len(reports), seqNr)
	return reports, nil
}

func (plugin *ocr3Plugin) ShouldAcceptAttestedReport(_ context.Context, seqNr uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	plugin.Logger.Printf("inside should accept attested report for sequence number %d", seqNr)
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	plugin.Logger.Printf("%d upkeeps found in report for should accept attested for sequence number %d", len(upkeeps), seqNr)

	accept := false
	// If any upkeep should be accepted, then accept
	// TODO: think through this logic
	for _, upkeep := range upkeeps {
		shouldAccept := plugin.Coordinator.ShouldAccept(upkeep)
		plugin.Logger.Printf("checking shouldAccept of upkeep '%s' in sequence number %d returned %t", upkeep.UpkeepID, seqNr, shouldAccept)

		if shouldAccept {
			accept = true
		}
	}

	return accept, nil
}

func (plugin *ocr3Plugin) ShouldTransmitAcceptedReport(_ context.Context, seqNr uint64, report ocr3types.ReportWithInfo[AutomationReportInfo]) (bool, error) {
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	plugin.Logger.Printf("%d upkeeps found in report for should transmit for sequence number %d", len(upkeeps), seqNr)

	transmit := false
	// If any upkeep should be transmitted, then transmit
	for _, upkeep := range upkeeps {
		shouldTransmit := plugin.Coordinator.ShouldTransmit(upkeep)
		plugin.Logger.Printf("checking transmit of upkeep '%s' in sequence number %d returned %t", upkeep.UpkeepID, seqNr, shouldTransmit)

		if shouldTransmit {
			transmit = true
		}
	}

	return transmit, nil
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

func (plugin *ocr3Plugin) getReportFromPerformables(toPerform []ocr2keepers.CheckResult) (ocr3types.ReportWithInfo[AutomationReportInfo], error) {
	encoded, err := plugin.ReportEncoder.Encode(toPerform...)
	return ocr3types.ReportWithInfo[AutomationReportInfo]{
		Report: types.Report(encoded),
	}, err
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
