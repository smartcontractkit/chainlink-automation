package keepers

import (
	"context"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	plugin := &keepers{}
	b, err := plugin.Query(context.Background(), types.ReportTimestamp{})

	assert.NoError(t, err)
	assert.Equal(t, types.Query{}, b)
}

func BenchmarkQuery(b *testing.B) {
	plugin := &keepers{}

	// run the Query function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.Query(context.Background(), types.ReportTimestamp{})
		if err != nil {
			b.Fail()
		}
	}
}

func TestObservation(t *testing.T) {
	t.Skip()
	plugin := &keepers{}
	b, err := plugin.Observation(context.Background(), types.ReportTimestamp{}, types.Query{})

	assert.NoError(t, err)
	assert.Equal(t, types.Observation{}, b)
}

func BenchmarkObservation(b *testing.B) {
	b.Skip()
	plugin := &keepers{}

	// run the Observation function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.Observation(context.Background(), types.ReportTimestamp{}, types.Query{})
		if err != nil {
			b.Fail()
		}
	}
}

func TestReport(t *testing.T) {
	t.Skip()
	plugin := &keepers{}
	ok, b, err := plugin.Report(context.Background(), types.ReportTimestamp{}, types.Query{}, []types.AttributedObservation{})

	assert.Equal(t, false, ok)
	assert.Equal(t, types.Report{}, b)
	assert.NoError(t, err)
}

func BenchmarkReport(b *testing.B) {
	b.Skip()
	plugin := &keepers{}

	// run the Report function b.N times
	for n := 0; n < b.N; n++ {
		_, _, err := plugin.Report(context.Background(), types.ReportTimestamp{}, types.Query{}, []types.AttributedObservation{})
		if err != nil {
			b.Fail()
		}
	}
}

func TestShouldAcceptFinalizedReport(t *testing.T) {
	plugin := &keepers{}
	ok, err := plugin.ShouldAcceptFinalizedReport(context.Background(), types.ReportTimestamp{}, types.Report{})

	assert.Equal(t, false, ok)
	assert.NoError(t, err)
}

func BenchmarkShouldAcceptFinalizedReport(b *testing.B) {
	plugin := &keepers{}

	// run the ShouldAcceptFinalizedReport function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.ShouldAcceptFinalizedReport(context.Background(), types.ReportTimestamp{}, types.Report{})
		if err != nil {
			b.Fail()
		}
	}
}

func TestShouldTransmitAcceptedReport(t *testing.T) {
	plugin := &keepers{}
	ok, err := plugin.ShouldTransmitAcceptedReport(context.Background(), types.ReportTimestamp{}, types.Report{})

	assert.Equal(t, false, ok)
	assert.NoError(t, err)
}

func BenchmarkShouldTransmitAcceptedReport(b *testing.B) {
	plugin := &keepers{}

	// run the ShouldTransmitAcceptedReport function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.ShouldTransmitAcceptedReport(context.Background(), types.ReportTimestamp{}, types.Report{})
		if err != nil {
			b.Fail()
		}
	}
}
