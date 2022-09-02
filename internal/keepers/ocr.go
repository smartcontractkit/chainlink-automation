package keepers

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (k *keepers) Query(_ context.Context, _ types.ReportTimestamp) (types.Query, error) {
	return types.Query{}, nil
}

func (k *keepers) Observation(ctx context.Context, _ types.ReportTimestamp, _ types.Query) (types.Observation, error) {
	// TODO: implement Observation using provided service
	_, err := k.service.SampleUpkeeps(ctx)
	return types.Observation{}, err
}

func (k *keepers) Report(ctx context.Context, _ types.ReportTimestamp, _ types.Query, _ []types.AttributedObservation) (bool, types.Report, error) {
	// TODO: implement Report using provided service
	_, err := k.service.CheckUpkeep(ctx, ktypes.UpkeepKey{})
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
