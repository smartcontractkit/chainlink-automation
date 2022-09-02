package keepers

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	k_types "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (k *keepers) Query(_ context.Context, _ types.ReportTimestamp) (types.Query, error) {
	return types.Query{}, nil
}

func (k *keepers) Observation(ctx context.Context, _ types.ReportTimestamp, _ types.Query) (types.Observation, error) {
	results, err := k.service.SampleUpkeeps(ctx)
	if err != nil {
		return types.Observation{}, err
	}

	keys := keyList(filterUpkeeps(results, Perform))

	b, err := encodeUpkeepKeys(keys)
	if err != nil {
		return types.Observation{}, err
	}

	return types.Observation(b), err
}

func (k *keepers) Report(ctx context.Context, _ types.ReportTimestamp, _ types.Query, _ []types.AttributedObservation) (bool, types.Report, error) {
	// TODO: implement Report using provided service
	_, err := k.service.CheckUpkeep(ctx, k_types.UpkeepKey{})
	return false, types.Report{}, err
}

func (k *keepers) ShouldAcceptFinalizedReport(_ context.Context, _ types.ReportTimestamp, _ types.Report) (bool, error) {
	return false, nil
}

func (k *keepers) ShouldTransmitAcceptedReport(c context.Context, t types.ReportTimestamp, r types.Report) (bool, error) {
	return false, nil
}

func (k *keepers) Close() error {
	return nil
}
