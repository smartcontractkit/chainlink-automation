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

	ob := observationMessageProto{
		RandomValue: k.rSrc.Int63(),
		Keys:        keyList(filterUpkeeps(results, ktypes.Eligible)),
	}

	b, err := encode(ob)
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
		// TODO: check if there is a lockout on the current id

		upkeep, err := k.service.CheckUpkeep(ctx, key)
		if err != nil {
			return false, nil, fmt.Errorf("%w: failed to check upkeep from attributed observation", err)
		}

		if upkeep.State == ktypes.Eligible {
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

	return true, types.Report(b), err
}

// ShouldAcceptFinalizedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation checks the length of the report and the number
// of keys in the report. Finally it applies a lockout to all keys in the report
func (k *keepers) ShouldAcceptFinalizedReport(ctx context.Context, rt types.ReportTimestamp, r types.Report) (bool, error) {
	if len(r) == 0 {
		k.logger.Printf("finalized report is empty; not accepting epoch %d and round %d", rt.Epoch, rt.Round)
		return false, nil
	}

	ids, err := k.encoder.IDsFromReport(r)
	if err != nil {
		return false, err
	}

	if len(ids) == 0 {
		k.logger.Printf("no upkeeps in report; not accepting epoch %d and round %d", rt.Epoch, rt.Round)
		return false, nil
	}

	for _, id := range ids {
		// set a lockout on the key
		err = k.service.LockoutUpkeep(ctx, id)
		if err != nil {
			return false, fmt.Errorf("failed to lock upkeep '%s' in epoch %d for round %d: %s", id, rt.Epoch, rt.Round, err)
		}
	}

	return true, nil
}

// ShouldTransmitAcceptedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation essentially draws straws on which node should
// be the transmitter.
func (k *keepers) ShouldTransmitAcceptedReport(ctx context.Context, rt types.ReportTimestamp, r types.Report) (bool, error) {
	ids, err := k.encoder.IDsFromReport(r)
	if err != nil {
		return false, fmt.Errorf("failed to get ids from report: %w", err)
	}

	if len(ids) == 0 {
		return false, fmt.Errorf("no ids in report in epoch %d for round %d", rt.Epoch, rt.Round)
	}

	for _, id := range ids {
		locked, err := k.service.IsUpkeepLocked(ctx, id)
		if err != nil {
			return false, fmt.Errorf("failed to check lock for upkeep '%s' in epoch %d for round %d: %s", string(id), rt.Epoch, rt.Round, err)
		}

		// multiple keys can be in a single report. if one fails to run in the
		// contract, but others are successful, don't try to transmit again
		if locked {
			k.logger.Printf("not transmitting report because upkeep '%s' is locked in epoch %d and round %d", string(id), rt.Epoch, rt.Round)
			return false, nil
		}
	}

	return true, nil
}

// Close implements the types.ReportingPlugin interface in OCR2.
func (k *keepers) Close() error {
	return nil
}
