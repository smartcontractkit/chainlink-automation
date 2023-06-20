package plugin

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type InstructionStore interface{}

type SamplingStore interface{}

type ocr3Plugin[RI any] struct {
	PrebuildHooks     []func(ocr2keepersv3.AutomationOutcome) error
	BuildHooks        []func(*ocr2keepersv3.AutomationObservation, InstructionStore, SamplingStore, ocr2keepersv3.ResultStore) error
	InstructionSource InstructionStore
	MetadataSource    SamplingStore
	ResultSource      ocr2keepersv3.ResultStore
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
			uid := fmt.Sprintf("%v", result.Payload)
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

func (plugin *ocr3Plugin[RI]) Reports(seqNr uint64, outcome ocr3types.Outcome) ([]ocr3types.ReportWithInfo[RI], error) {
	panic("ocr3 Reports not implemented")
}

func (plugin *ocr3Plugin[RI]) ShouldAcceptFinalizedReport(context.Context, uint64, ocr3types.ReportWithInfo[RI]) (bool, error) {
	panic("ocr3 ShouldAcceptFinalizedReport not implemented")
}

func (plugin *ocr3Plugin[RI]) ShouldTransmitAcceptedReport(context.Context, uint64, ocr3types.ReportWithInfo[RI]) (bool, error) {
	panic("ocr3 ShouldTransmitAcceptedReport not implemented")
}

func (plugin *ocr3Plugin[RI]) Close() error {
	panic("ocr3 Close not implemented")
}
