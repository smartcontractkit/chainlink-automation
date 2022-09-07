package keepers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
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
		SampleSet           []*ktypes.UpkeepResult
		SampleErr           error
		ExpectedObservation types.Observation
		ExpectedErr         error
	}{
		{
			Name:                "Empty Set",
			Ctx:                 func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:           []*ktypes.UpkeepResult{},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{})),
		},
		{
			Name:                "Timer Context",
			Ctx:                 func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			SampleSet:           []*ktypes.UpkeepResult{},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{})),
		},
		{
			Name:                "Upkeep Service Error",
			Ctx:                 func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:           []*ktypes.UpkeepResult{},
			SampleErr:           fmt.Errorf("test error"),
			ExpectedObservation: types.Observation{},
			ExpectedErr:         fmt.Errorf("test error"),
		},
		{
			Name: "Filter to Empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*ktypes.UpkeepResult{
				{Key: ktypes.UpkeepKey([]byte("1|1")), State: Skip},
				{Key: ktypes.UpkeepKey([]byte("1|2")), State: Skip},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{})),
		},
		{
			Name: "Filter to Non-empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*ktypes.UpkeepResult{
				{Key: ktypes.UpkeepKey([]byte("1|1")), State: Skip},
				{Key: ktypes.UpkeepKey([]byte("1|2")), State: Perform},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{[]byte("1|2")})),
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

	set := make([]*ktypes.UpkeepResult, 2, 100)
	set[0] = &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte("1|1")), State: Perform}
	set[1] = &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte("1|2")), State: Perform}

	for i := 2; i < 100; i++ {
		set = append(set, &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1))), State: Skip})
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
	tests := []struct {
		Name         string
		Ctx          func() (context.Context, func())
		Observations []types.AttributedObservation
		Checks       []struct {
			K ktypes.UpkeepKey
			R ktypes.UpkeepResult
			E error
		}
		Perform        []int
		EncodeErr      error
		ExpectedReport []byte
		ExpectedBool   bool
		ExpectedErr    error
	}{
		{
			Name: "Single Common Upkeep",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")}},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name: "Forward Context",
			Ctx:  func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{
					K: ktypes.UpkeepKey([]byte("1|1")),
					R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name: "Wrap Error",
			Ctx:  func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{}, E: ErrMockTestError},
			},
			ExpectedBool: false,
			ExpectedErr:  ErrMockTestError,
		},
		{
			Name: "Unsorted Observations",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2")), ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")}},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name: "Earliest Block",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("2|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("3|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")}},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name: "Skip Already Performed",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1")), ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Reported, PerformData: []byte("abcd")}},
				{K: ktypes.UpkeepKey([]byte("1|2")), R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")}},
			},
			Perform:        []int{1},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name: "Nothing to Report",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1")), ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Reported, PerformData: []byte("abcd")}},
				{K: ktypes.UpkeepKey([]byte("1|2")), R: ktypes.UpkeepResult{State: Reported, PerformData: []byte("abcd")}},
			},
			ExpectedBool: false,
		},
		{
			Name: "Empty Observations",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{}))},
			},
			ExpectedBool: false,
		},
		{
			Name:         "No Observations",
			Ctx:          func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{},
			ExpectedBool: false,
			ExpectedErr:  ErrNotEnoughInputs,
		},
		{
			Name: "Encoding Error",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K ktypes.UpkeepKey
				R ktypes.UpkeepResult
				E error
			}{
				{K: ktypes.UpkeepKey([]byte("1|1")), R: ktypes.UpkeepResult{State: Perform, PerformData: []byte("abcd")}},
			},
			Perform:      []int{0},
			EncodeErr:    ErrMockTestError,
			ExpectedBool: false,
			ExpectedErr:  ErrMockTestError,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ms := new(MockedUpkeepService)
			me := new(MockedReportEncoder)

			plugin := &keepers{service: ms, encoder: me}
			ctx, cancel := test.Ctx()

			// set up upkeep checks with the mocked service
			for _, check := range test.Checks {
				check.R.Key = check.K
				ms.Mock.On("CheckUpkeep", ctx, check.K).Return(check.R, check.E)
			}

			if len(test.Perform) > 0 {
				toPerform := make([]ktypes.UpkeepResult, len(test.Perform))
				for i, p := range test.Perform {
					u := test.Checks[p]
					u.R.Key = u.K
					toPerform[i] = u.R
				}
				me.Mock.On("EncodeReport", toPerform).Return(test.ExpectedReport, test.EncodeErr)
			}

			// test the Report function
			ok, r, err := plugin.Report(ctx, types.ReportTimestamp{}, types.Query{}, test.Observations)
			cancel()

			assert.Equal(t, test.ExpectedBool, ok)
			assert.Equal(t, types.Report(test.ExpectedReport), r)

			if test.ExpectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, test.ExpectedErr)
			}

			ms.Mock.AssertExpectations(t)
			me.Mock.AssertExpectations(t)
		})
	}
}

func BenchmarkReport(b *testing.B) {
	ms := new(MockedUpkeepService)
	me := new(MockedReportEncoder)
	plugin := &keepers{service: ms, encoder: me}

	set := ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte("1|1")), State: Perform, PerformData: []byte("abcd")}
	observations := []types.AttributedObservation{
		{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
		{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
		{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
	}

	b.ResetTimer()
	// run the Observation function b.N times
	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		ms.Mock.On("CheckUpkeep", ctx, set.Key).Return(set, nil)
		me.Mock.On("EncodeReport", []ktypes.UpkeepResult{set}).Return([]byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))), nil)

		b.StartTimer()
		_, _, err := plugin.Report(context.Background(), types.ReportTimestamp{}, types.Query{}, observations)
		b.StopTimer()

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

var (
	ErrMockTestError = fmt.Errorf("test error")
)

type MockedUpkeepService struct {
	mock.Mock
}

func (_m *MockedUpkeepService) SampleUpkeeps(ctx context.Context) ([]*ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx)

	var r0 []*ktypes.UpkeepResult
	if rf, ok := ret.Get(0).(func() []*ktypes.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*ktypes.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedUpkeepService) CheckUpkeep(ctx context.Context, key ktypes.UpkeepKey) (ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 ktypes.UpkeepResult
	if rf, ok := ret.Get(0).(func() ktypes.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ktypes.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedUpkeepService) SetUpkeepState(ctx context.Context, key ktypes.UpkeepKey, state ktypes.UpkeepState) error {
	return _m.Mock.Called(ctx, key, state).Error(0)
}

func mustEncodeKeys(keys []ktypes.UpkeepKey) []byte {
	b, _ := Encode(keys)
	return b
}

type MockedReportEncoder struct {
	mock.Mock
}

func (_m *MockedReportEncoder) EncodeReport(toReport []ktypes.UpkeepResult) ([]byte, error) {
	ret := _m.Mock.Called(toReport)

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0, ret.Error(1)
}
