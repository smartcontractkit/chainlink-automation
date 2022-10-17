package keepers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type ocrLogContextKey struct{}

type ocrLogContext struct {
	Epoch     uint32
	Round     uint8
	StartTime time.Time
}

func newOcrLogContext(rt types.ReportTimestamp) ocrLogContext {
	return ocrLogContext{
		Epoch:     rt.Epoch,
		Round:     rt.Round,
		StartTime: time.Now(),
	}
}

func (c ocrLogContext) String() string {
	return fmt.Sprintf("[epoch=%d, round=%d, completion=%dms]", c.Epoch, c.Round, time.Since(c.StartTime)/time.Millisecond)
}

func (c ocrLogContext) Short() string {
	return fmt.Sprintf("[epoch=%d, round=%d]", c.Epoch, c.Round)
}

// Query implements the types.ReportingPlugin interface in OCR2. The query produced from this
// method is intended to be empty.
func (k *keepers) Query(_ context.Context, _ types.ReportTimestamp) (types.Query, error) {
	return types.Query{}, nil
}

// Observation implements the types.ReportingPlugin interface in OCR2. This method samples a set
// of upkeeps available in and UpkeepService and produces an observation containing upkeeps that
// need to be executed.
func (k *keepers) Observation(ctx context.Context, rt types.ReportTimestamp, _ types.Query) (types.Observation, error) {
	lCtx := newOcrLogContext(rt)
	ctx = context.WithValue(ctx, ocrLogContextKey{}, lCtx)

	results, err := k.service.SampleUpkeeps(ctx, k.filter.Filter())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to sample upkeeps for observation: %s", err, lCtx)
	}

	keys := keyList(filterUpkeeps(results, ktypes.Eligible))

	// limit the number of keys that can be added to an observation
	// OCR observation limit is set to 1_000 bytes so this should be under the
	// limit
	if len(keys) > 10 {
		keys = keys[:10]
	}

	b, err := encode(keys)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode upkeep keys for observation: %s", err, lCtx)
	}

	// write the number of keys returned from sampling to the debug log
	// this offers a record of the number of performs the node has visibility
	// of for each epoch/round
	k.logger.Printf("OCR observation completed successfully with %d eligible keys: %s", len(keys), lCtx)

	return types.Observation(b), nil
}

// Report implements the types.ReportingPlugin inteface in OC2. This method chooses a single upkeep
// from the provided observations by the earliest block number, checks the upkeep, and builds a
// report. Multiple upkeeps in a single report is supported by how the data is abi encoded, but
// no gas estimations exist yet.
func (k *keepers) Report(ctx context.Context, rt types.ReportTimestamp, _ types.Query, attributed []types.AttributedObservation) (bool, types.Report, error) {
	var err error

	lCtx := newOcrLogContext(rt)
	ctx = context.WithValue(ctx, ocrLogContextKey{}, lCtx)

	// pass the filter to the dedupe function
	// ensure no locked keys come through
	keys, err := sortedDedupedKeyList(attributed, k.filter.Filter())
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to sort/dedupe attributed observations: %s", err, lCtx)
	}

	// select, verify, and build report
	toPerform := make([]ktypes.UpkeepResult, 0, 1)
	for _, key := range keys {

		upkeep, err := k.service.CheckUpkeep(ctx, key)
		if err != nil {
			return false, nil, fmt.Errorf("%w: failed to check upkeep from attributed observation: %s", err, lCtx)
		}

		if upkeep.State == ktypes.Eligible {
			// only build a report from a single upkeep for now
			k.logger.Printf("reporting %s to be performed: %s", upkeep.Key, lCtx.Short())
			toPerform = append(toPerform, upkeep)
			break
		}
	}

	// if nothing to report, return false with no error
	if len(toPerform) == 0 {
		k.logger.Printf("OCR report completed successfully with no upkeeps added to the report: %s", lCtx)
		return false, nil, nil
	}

	b, err := k.encoder.EncodeReport(toPerform)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to encode OCR report: %s", err, lCtx)
	}

	k.logger.Printf("OCR report completed successfully with %d upkeep added to the report: %s", len(toPerform), lCtx)

	return true, types.Report(b), err
}

// ShouldAcceptFinalizedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation checks the length of the report and the number
// of keys in the report. Finally it applies a lockout to all keys in the report
func (k *keepers) ShouldAcceptFinalizedReport(_ context.Context, rt types.ReportTimestamp, r types.Report) (bool, error) {
	lCtx := newOcrLogContext(rt)

	if len(r) == 0 {
		k.logger.Printf("finalized report is empty; not accepting: %s", lCtx)
		return false, nil
	}

	results, err := k.encoder.DecodeReport(r)
	if err != nil {
		return false, fmt.Errorf("%w: failed to encode report: %s", err, lCtx)
	}

	if len(results) == 0 {
		k.logger.Printf("no upkeeps in report; not accepting: %s", lCtx)
		return false, nil
	}

	for _, r := range results {
		// indicate to the filter that the key has been accepted for transmit
		err = k.filter.Accept(r.Key)
		if err != nil {
			if errors.Is(err, ErrKeyAlreadySet) {
				k.logger.Printf("%s: key already set: %s", r.Key, lCtx.Short())
				return false, nil
			}
			return false, fmt.Errorf("%w: failed to accept key: %s", err, lCtx)
		}
		k.logger.Printf("accepting key %s: %s", r.Key, lCtx.Short())
	}

	k.logger.Printf("OCR should accept completed successfully: %s", lCtx)

	return true, nil
}

// ShouldTransmitAcceptedReport implements the types.ReportingPlugin interface
// from OCR2. The implementation essentially draws straws on which node should
// be the transmitter.
func (k *keepers) ShouldTransmitAcceptedReport(_ context.Context, rt types.ReportTimestamp, r types.Report) (bool, error) {
	lCtx := newOcrLogContext(rt)

	results, err := k.encoder.DecodeReport(r)
	if err != nil {
		return false, fmt.Errorf("%w: failed to get ids from report: %s", err, lCtx)
	}

	if len(results) == 0 {
		return false, fmt.Errorf("no ids in report: %s", lCtx)
	}

	for _, id := range results {
		transmitConfirmed := k.filter.IsTransmissionConfirmed(id.Key)
		// multiple keys can be in a single report. if one has a confirmed transmission
		// (while others may not have), don't try to transmit again
		// TODO: reevaluate this assumption
		if transmitConfirmed {
			k.logger.Printf("not transmitting report because upkeep '%s' was already transmitted: %s", string(id.Key), lCtx)
			return false, nil
		}

		k.logger.Printf("upkeep '%s' transmit not confirmed: %s", id.Key, lCtx.Short())
	}

	k.logger.Printf("OCR should transmit completed successfully: %s", lCtx)

	return true, nil
}

// Close implements the types.ReportingPlugin interface in OCR2.
func (k *keepers) Close() error {
	return nil
}
