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

	rdm, keys, err := sortedDedupedKeyList(attributed)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to sort/dedupe attributed observations", err)
	}

	var sent int64
	values := make([]int64, len(rdm))
	for i, r := range rdm {
		values[i] = r.Value
		if r.Observer == k.id {
			sent = r.Value
		}
	}

	// like drawing straws, if the lowest value from all nodes was sent by this
	// node, this node should be the next transmitter
	isTransmitter := sent == lowest(values)
	k.logger.Printf("node selected as transmitter: %t", isTransmitter)
	k.mu.Lock()
	k.transmit = isTransmitter
	k.mu.Unlock()

	// select, verify, and build report
	toPerform := make([]ktypes.UpkeepResult, 0, 1)
	for _, key := range keys {
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

	// TODO: move this to ShouldAcceptFinalizedReport
	// update internal state of upkeeps to ensure they aren't reported or observed again
	for i := 0; i < len(toPerform); i++ {
		if err := k.service.SetUpkeepState(ctx, toPerform[i].Key, ktypes.InFlight); err != nil {
			return false, nil, fmt.Errorf("%w: failed to update internal state while generating report", err)
		}
	}

	return true, types.Report(b), err
}

// ShouldAcceptFinalizedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation is the most basic possible in that it only
// checks that a report has data to send.
func (k *keepers) ShouldAcceptFinalizedReport(_ context.Context, _ types.ReportTimestamp, r types.Report) (bool, error) {
	// TODO: decode report, set reported status for each upkeep
	// TODO: isStale check on last performed block number
	shouldAccept := len(r) != 0
	if shouldAccept {
		k.logger.Print("accepting finalized report")
	} else {
		k.logger.Print("finalized report empty; not accepting")
	}
	return shouldAccept, nil
}

// ShouldTransmitAcceptedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation essentially draws straws on which node should
// be the transmitter.
func (k *keepers) ShouldTransmitAcceptedReport(_ context.Context, _ types.ReportTimestamp, _ types.Report) (bool, error) {
	// TODO: isStale check on last performed block number
	// TODO: check that transmission has not completed
	var isTransmitter bool
	k.mu.Lock()
	isTransmitter = k.transmit
	k.mu.Unlock()

	if isTransmitter {
		k.logger.Print("accepting report for transmit")
	} else {
		k.logger.Print("report available for transmit; not a transmitter")
	}

	return isTransmitter, nil
}

// Close implements the types.ReportingPlugin interface in OCR2.
func (k *keepers) Close() error {
	return nil
}
