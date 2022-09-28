package keepers

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// Query implements the types.ReportingPlugin interface in OCR2. The query produced from this
// method is intended to be empty.
func (k *keepers) Query(_ context.Context, _ types.ReportTimestamp) (types.Query, error) {
	return types.Query{}, nil
}

// Observation implements the types.ReportingPlugin interface in OCR2. This method samples a set
// of upkeeps available in and UpkeepService and produces an observation containing upkeeps that
// need to be executed.
func (k *keepers) Observation(ctx context.Context, _ types.ReportTimestamp, _ types.Query) (types.Observation, error) {
	results, err := k.service.SampleUpkeeps(ctx)
	if err != nil {
		return types.Observation{}, err
	}

	keys := keyList(filterUpkeeps(results, ktypes.Perform))

	b, err := encode(keys)
	if err != nil {
		return types.Observation{}, err
	}

	return types.Observation(b), err
}

// Report implements the types.ReportingPlugin inteface in OC2. This method chooses a single upkeep
// from the provided observations by the earliest block number, checks the upkeep, and builds a
// report. Multiple upkeeps in a single report is supported by how the data is abi encoded, but
// no gas estimations exist yet.
func (k *keepers) Report(ctx context.Context, _ types.ReportTimestamp, _ types.Query, attributed []types.AttributedObservation) (bool, types.Report, error) {
	var err error

	keys, err := sortedDedupedKeyList(attributed)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to sort/dedupe attributed observations", err)
	}

	// select, verify, and build report
	toPerform := make([]ktypes.UpkeepResult, 0, 1)
	for _, key := range keys {
		upkeep, err := k.service.CheckUpkeep(ctx, key)
		if err != nil {
			return false, nil, fmt.Errorf("%w: failed to check upkeep from attributed observation", err)
		}

		if upkeep.State == ktypes.Perform {
			// only build a report from a single upkeep for now
			k.logger.Printf("reporting %s to be performed", upkeep.Key)
			toPerform = append(toPerform, upkeep)
			break
		}
	}

	// if nothing to report, return false with no error
	if len(toPerform) == 0 {
		return false, nil, nil
	}

	b, err := k.encoder.EncodeReport(toPerform)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to encode OCR report", err)
	}

	// update internal state of upkeeps to ensure they aren't reported or observed again
	for i := 0; i < len(toPerform); i++ {
		if err := k.service.SetUpkeepState(ctx, toPerform[i].Key, ktypes.Reported); err != nil {
			return false, nil, fmt.Errorf("%w: failed to update internal state while generating report", err)
		}
	}

	return true, types.Report(b), err
}

// ShouldAcceptFinalizedReport implements the types.ReportingPlugin interface in OCR2. The implementation
// is the most basic possible at this point and assumes all reports should be accepted.
func (k *keepers) ShouldAcceptFinalizedReport(_ context.Context, _ types.ReportTimestamp, _ types.Report) (bool, error) {
	// TODO: add some checks to verify that a report should be accepted to transmit
	return true, nil
}

// ShouldTransmitAcceptedReport implements the types.ReportingPlugin interface in OCR2. The implementation
// is the most basic possible at this point and assumes all reports should be accepted.
func (k *keepers) ShouldTransmitAcceptedReport(c context.Context, t types.ReportTimestamp, r types.Report) (bool, error) {
	// TODO: add some checks to verify that a report should be accepted to transmit
	return true, nil
}

// Close implements the types.ReportingPlugin interface in OCR2.
func (k *keepers) Close() error {
	return nil
}
