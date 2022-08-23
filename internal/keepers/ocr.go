package keepers

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func (k *keepers) Query(_ context.Context, _ types.ReportTimestamp) (types.Query, error) {
	return types.Query{}, nil
}

func (k *keepers) Observation(ctx context.Context, _ types.ReportTimestamp, _ types.Query) (types.Observation, error) {
	return types.Observation{}, nil
}

func (k *keepers) Report(_ context.Context, _ types.ReportTimestamp, _ types.Query, _ []types.AttributedObservation) (bool, types.Report, error) {
	return false, types.Report{}, nil
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
