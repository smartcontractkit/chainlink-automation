package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
)

type Encoder interface {
	Encode(...ocr2keepers.CheckResult) ([]byte, error)
	Extract([]byte) ([]ocr2keepers.ReportedUpkeep, error)
}

type Coordinator interface {
	IsTransmissionConfirmed(ocr2keepers.ReportedUpkeep) bool
}

type ocr3Plugin[RI any] struct {
	PrebuildHooks      []func(ocr2keepersv3.AutomationOutcome) error
	BuildHooks         []func(*ocr2keepersv3.AutomationObservation, ocr2keepersv3.InstructionStore, ocr2keepersv3.MetadataStore, ocr2keepersv3.ResultStore) error
	InstructionSource  ocr2keepersv3.InstructionStore
	MetadataSource     ocr2keepersv3.MetadataStore
	ResultSource       ocr2keepersv3.ResultStore
	ReportEncoder      Encoder
	Coordinator        Coordinator
	Services           []service.Recoverable
	ReportGasLimit     uint64
	MaxUpkeepBatchSize int
}

func (plugin *ocr3Plugin[RI]) Query(ctx context.Context, outctx ocr3types.OutcomeContext) (types.Query, error) {
	panic("ocr3 Query not implemented")
}

func (plugin *ocr3Plugin[RI]) Observation(ctx context.Context, outcome ocr3types.OutcomeContext, query types.Query) (types.Observation, error) {
	// Decode the outcome to AutomationOutcome
	automationOutcome, err := ocr2keepersv3.DecodeAutomationOutcome(outcome.PreviousOutcome)
	if err != nil {
		return nil, err
	}

	// Execute pre-build hooks
	for _, hook := range plugin.PrebuildHooks {
		err = errors.Join(err, hook(automationOutcome))
	}
	if err != nil {
		return nil, err
	}

	// Create new AutomationObservation
	observation := ocr2keepersv3.AutomationObservation{}

	// Execute build hooks
	for _, hook := range plugin.BuildHooks {
		err := hook(&observation, plugin.InstructionSource, plugin.MetadataSource, plugin.ResultSource)
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

func (plugin *ocr3Plugin[RI]) ValidateObservation(outctx ocr3types.OutcomeContext, query types.Query, ao types.AttributedObservation) error {
	panic("ocr3 ValidateObservation not implemented")
}

func (plugin *ocr3Plugin[RI]) Outcome(outctx ocr3types.OutcomeContext, query types.Query, attributedObservations []types.AttributedObservation) (ocr3types.Outcome, error) {
	type resultAndCount struct {
		result ocr2keepers.CheckResult
		count  int
	}

	resultCount := make(map[string]resultAndCount)

	for _, attributedObservation := range attributedObservations {
		observation, err := ocr2keepersv3.DecodeAutomationObservation(attributedObservation.Observation)
		if err != nil {
			return nil, err
		}

		for _, result := range observation.Performable {
			uid := fmt.Sprintf("%v", result)
			payloadCount, ok := resultCount[uid]
			if !ok {
				payloadCount = resultAndCount{
					result: result,
					count:  1,
				}
			} else {
				payloadCount.count++
			}
			resultCount[uid] = payloadCount
		}
	}

	submittedObservations := len(attributedObservations)
	quorumThreshold := submittedObservations / 2

	var performable []ocr2keepers.CheckResult
	for _, payload := range resultCount {
		if payload.count > quorumThreshold {
			performable = append(performable, payload.result)
		}
	}

	outcome := ocr2keepersv3.AutomationOutcome{
		Performable: performable,
	}

	return outcome.Encode()
}

func (plugin *ocr3Plugin[RI]) Reports(_ uint64, raw ocr3types.Outcome) ([]ocr3types.ReportWithInfo[RI], error) {
	var (
		reports []ocr3types.ReportWithInfo[RI]
		outcome ocr2keepersv3.AutomationOutcome
		err     error
	)

	if outcome, err = ocr2keepersv3.DecodeAutomationOutcome(raw); err != nil {
		return nil, err
	}

	toPerform := []ocr2keepers.CheckResult{}
	var gasUsed uint64

	for i, result := range outcome.Performable {
		if gasUsed+result.GasUsed > plugin.ReportGasLimit || len(toPerform) > plugin.MaxUpkeepBatchSize {
			// encode current collection
			encoded, encodeErr := plugin.ReportEncoder.Encode(toPerform...)
			err = errors.Join(err, encodeErr)

			if encodeErr == nil {
				// add encoded data to reports
				reports = append(reports, ocr3types.ReportWithInfo[RI]{
					Report: types.Report(encoded),
				})

				// reset collection
				toPerform = []ocr2keepers.CheckResult{}
				gasUsed = 0
			}
		}

		gasUsed += result.GasUsed
		toPerform = append(toPerform, outcome.Performable[i])
	}

	// if there are still values to add
	if len(toPerform) > 0 {
		// encode current collection
		encoded, encodeErr := plugin.ReportEncoder.Encode(toPerform...)
		err = errors.Join(err, encodeErr)

		if encodeErr == nil {
			// add encoded data to reports
			reports = append(reports, ocr3types.ReportWithInfo[RI]{
				Report: types.Report(encoded),
			})
		}
	}

	return reports, err
}

func (plugin *ocr3Plugin[RI]) ShouldAcceptFinalizedReport(context.Context, uint64, ocr3types.ReportWithInfo[RI]) (bool, error) {
	panic("ocr3 ShouldAcceptFinalizedReport not implemented")
}

func (plugin *ocr3Plugin[RI]) ShouldTransmitAcceptedReport(_ context.Context, _ uint64, report ocr3types.ReportWithInfo[RI]) (bool, error) {
	upkeeps, err := plugin.ReportEncoder.Extract(report.Report)
	if err != nil {
		return false, err
	}

	for _, upkeep := range upkeeps {
		if !plugin.Coordinator.IsTransmissionConfirmed(upkeep) {
			// if any upkeep in the report does not have confirmation, attempt
			// again
			return true, nil
		}
	}

	return false, nil
}

func (plugin *ocr3Plugin[RI]) Close() error {
	var err error

	for i := range plugin.Services {
		err = errors.Join(err, plugin.Services[i].Close())
	}

	return err
}

func (plugin *ocr3Plugin[RI]) startServices() {
	for i := range plugin.Services {
		go func(svc service.Recoverable) {
			_ = svc.Start(context.Background())
		}(plugin.Services[i])
	}
}
