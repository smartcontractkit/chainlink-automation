package keepers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	k_types "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	tests := []struct {
		Name                string
		Ctx                 func() (context.Context, func())
		SampleSet           []*k_types.UpkeepResult
		SampleErr           error
		ExpectedObservation types.Observation
		ExpectedErr         error
	}{
		{
			Name:                "Empty Set",
			Ctx:                 func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:           []*k_types.UpkeepResult{},
			ExpectedObservation: types.Observation(mustEncodeKeys([]k_types.UpkeepKey{})),
		},
		{
			Name:                "Timer Context",
			Ctx:                 func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			SampleSet:           []*k_types.UpkeepResult{},
			ExpectedObservation: types.Observation(mustEncodeKeys([]k_types.UpkeepKey{})),
		},
		{
			Name:                "Upkeep Service Error",
			Ctx:                 func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:           []*k_types.UpkeepResult{},
			SampleErr:           fmt.Errorf("test error"),
			ExpectedObservation: types.Observation{},
			ExpectedErr:         fmt.Errorf("test error"),
		},
		{
			Name: "Filter to Empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*k_types.UpkeepResult{
				{Key: k_types.UpkeepKey([]byte("1|1")), State: Skip},
				{Key: k_types.UpkeepKey([]byte("1|2")), State: Skip},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]k_types.UpkeepKey{})),
		},
		{
			Name: "Filter to Non-empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*k_types.UpkeepResult{
				{Key: k_types.UpkeepKey([]byte("1|1")), State: Skip},
				{Key: k_types.UpkeepKey([]byte("1|2")), State: Perform},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]k_types.UpkeepKey{[]byte("1|2")})),
		},
	}

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ms := new(MockedUpkeepService)
			plugin := &keepers{service: ms}

			ctx, cancel := test.Ctx()
			ms.Mock.On("SampleUpkeeps", ctx).Return(test.SampleSet, test.SampleErr)

			b, err := plugin.Observation(ctx, types.ReportTimestamp{}, types.Query{})
			cancel()

			if test.ExpectedErr == nil {
				assert.NoError(t, err, "no error expected for test %d; got %s", i+1, err)
			} else {
				assert.Equal(t, err.Error(), test.ExpectedErr.Error(), "error should match expected for test %d", i+1)
			}

			assert.Equal(t, test.ExpectedObservation, b, "observation mismatch for test %d", i+1)

			// assert that the context passed to Observation is also passed to the service
			ms.Mock.AssertExpectations(t)
		})
	}
}

func BenchmarkObservation(b *testing.B) {
	ms := new(MockedUpkeepService)
	plugin := &keepers{service: ms}

	set := make([]*k_types.UpkeepResult, 2, 100)
	set[0] = &k_types.UpkeepResult{Key: k_types.UpkeepKey([]byte("1|1")), State: Perform}
	set[1] = &k_types.UpkeepResult{Key: k_types.UpkeepKey([]byte("1|2")), State: Perform}

	for i := 2; i < 100; i++ {
		set = append(set, &k_types.UpkeepResult{Key: k_types.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1))), State: Skip})
	}

	b.ResetTimer()
	// run the Observation function b.N times
	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		ms.Mock.On("SampleUpkeeps", ctx).Return(set, nil)

		b.StartTimer()
		_, err := plugin.Observation(ctx, types.ReportTimestamp{}, types.Query{})
		b.StopTimer()

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

type MockedUpkeepService struct {
	mock.Mock
}

func (_m *MockedUpkeepService) SampleUpkeeps(ctx context.Context) ([]*k_types.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx)

	var r0 []*k_types.UpkeepResult
	if rf, ok := ret.Get(0).(func() []*k_types.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*k_types.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedUpkeepService) CheckUpkeep(ctx context.Context, key k_types.UpkeepKey) (k_types.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 k_types.UpkeepResult
	if rf, ok := ret.Get(0).(func() k_types.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(k_types.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedUpkeepService) SetUpkeepState(ctx context.Context, key k_types.UpkeepKey, state k_types.UpkeepState) error {
	return _m.Mock.Called(ctx, key, state).Error(0)
}

func mustEncodeKeys(keys []k_types.UpkeepKey) []byte {
	b, _ := encodeUpkeepKeys(keys)
	return b
}
