package keepers

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"golang.org/x/crypto/sha3"
)

const (
	// observationKeysLimit is the max number of keys that Observation could return.
	observationKeysLimit = 1

	// reportKeysLimit is the maximum number of upkeep keys returned by the report phase
	reportKeysLimit = 10
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

	// keyList produces a sorted result so the following reduction of keys
	// should be more uniform for all nodes
	keys := keyList(filterUpkeeps(results, ktypes.Eligible))

	// Check limit
	if len(keys) > observationKeysLimit {
		keys = keys[:observationKeysLimit]
	}

	latestBlock, err := k.service.LatestBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get latest block", err)
	}

	obs := &ktypes.UpkeepObservation{
		BlockKey:          ktypes.BlockKey(latestBlock.String()),
		UpkeepIdentifiers: []ktypes.UpkeepIdentifier{},
	}

	if len(keys) > 0 {
		var identifiers []ktypes.UpkeepIdentifier
		for _, upkeepKey := range keys {
			_, upkeepID, _ := upkeepKey.BlockKeyAndUpkeepID()
			identifiers = append(identifiers, upkeepID)
		}
		obs.UpkeepIdentifiers = identifiers
	}

	b, err := limitedLengthEncode(obs, maxObservationLength)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode upkeep keys for observation: %s", err, lCtx)
	}

	// write the number of keys returned from sampling to the debug log
	// this offers a record of the number of performs the node has visibility
	// of for each epoch/round
	k.logger.Printf("OCR observation completed successfully with %d eligible keys: %s", len(keys), lCtx)

	return b, nil
}

// Report implements the types.ReportingPlugin interface in OC2. This method chooses a single upkeep
// from the provided observations by the earliest block number, checks the upkeep, and builds a
// report. Multiple upkeeps in a single report is supported by how the data is abi encoded, but
// no gas estimations exist yet.
func (k *keepers) Report(ctx context.Context, rt types.ReportTimestamp, _ types.Query, attributed []types.AttributedObservation) (bool, types.Report, error) {
	var err error

	lCtx := newOcrLogContext(rt)
	ctx = context.WithValue(ctx, ocrLogContextKey{}, lCtx)

	// similar key building as libocr transmit selector
	hash := sha3.NewLegacyKeccak256()
	hash.Write(rt.ConfigDigest[:])
	temp := make([]byte, 8)
	binary.LittleEndian.PutUint64(temp, uint64(rt.Epoch))
	hash.Write(temp)
	binary.LittleEndian.PutUint64(temp, uint64(rt.Round))
	hash.Write(temp)

	var key [16]byte
	copy(key[:], hash.Sum(nil))

	// pass the filter to the dedupe function
	// ensure no locked keys come through
	keys, err := shuffleUniqueObservations(attributed, key, reportKeysLimit, k.filter.Filter())
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to sort/dedupe attributed observations: %s", err, lCtx)
	}

	// No keys found for the given keys
	if len(keys) == 0 {
		k.logger.Printf("OCR report completed successfully with no eligible keys: %s", lCtx)
		return false, nil, nil
	}

	// Check all upkeeps from the given observation
	checkedUpkeeps, err := k.service.CheckUpkeep(ctx, keys...)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to check upkeeps from attributed observation: %s", err, lCtx)
	}

	// No upkeeps found for the given keys
	if len(checkedUpkeeps) == 0 {
		k.logger.Printf("OCR report completed successfully with no successfully checked upkeeps: %s", lCtx)
		return false, nil, nil
	}

	if len(checkedUpkeeps) > len(keys) {
		return false, nil, fmt.Errorf("unexpected number of upkeeps returned for %s key, expected max %d but given %d", key, len(keys), len(checkedUpkeeps))
	}

	// Collect eligible upkeeps
	var reportCapacity uint32
	toPerform := make([]ktypes.UpkeepResult, 0, len(checkedUpkeeps))
	for _, checkedUpkeep := range checkedUpkeeps {
		if checkedUpkeep.State != ktypes.Eligible {
			continue
		}

		upkeepMaxGas := checkedUpkeep.ExecuteGas + k.upkeepGasOverhead
		if reportCapacity+upkeepMaxGas > k.reportGasLimit {
			// We don't break here since there could be an upkeep with the lower
			// gas limit so there could be a space for it in the report.
			k.logger.Printf("skipping upkeep %s due to report limit, current capacity is %d, upkeep gas is %d with %d overhead", checkedUpkeep.Key, reportCapacity, checkedUpkeep.ExecuteGas, k.upkeepGasOverhead)
			continue
		}

		k.logger.Printf("reporting %s to be performed with gas limit %d and %d overhead: %s", checkedUpkeep.Key, checkedUpkeep.ExecuteGas, k.upkeepGasOverhead, lCtx.Short())

		toPerform = append(toPerform, checkedUpkeep)
		reportCapacity += upkeepMaxGas

		// Don't exceed specified maxUpkeepBatchSize value in offchain config
		if len(toPerform) >= k.maxUpkeepBatchSize {
			break
		}
	}

	// if nothing to report, return false with no error
	if len(toPerform) == 0 {
		k.logger.Printf("OCR report completed successfully with no eligible upkeeps: %s", lCtx)
		return false, nil, nil
	}

	b, err := k.encoder.EncodeReport(toPerform)
	if err != nil {
		return false, nil, fmt.Errorf("%w: failed to encode OCR report: %s", err, lCtx)
	}

	k.logger.Printf("OCR report completed successfully with %d upkeep added to the report: %s", len(toPerform), lCtx)

	return true, b, err
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
		return false, fmt.Errorf("%w: failed to decode report: %s", err, lCtx)
	}

	if len(results) == 0 {
		k.logger.Printf("no upkeeps in report; not accepting: %s", lCtx)
		return false, fmt.Errorf("no ids in report: %s", lCtx)
	}

	for _, r := range results {
		// check whether the key is already accepted
		if k.filter.CheckAlreadyAccepted(r.Key) {
			k.logger.Printf("%s: key already accepted: %s", r.Key, lCtx.Short())
			return false, fmt.Errorf("failed to accept key %s as it was already accepted", r.Key)
		}
	}

	for _, r := range results {
		// indicate to the filter that the key has been accepted for transmit
		if err = k.filter.Accept(r.Key); err != nil {
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
			k.logger.Printf("not transmitting report because upkeep '%s' was already transmitted: %s", id.Key, lCtx)
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
