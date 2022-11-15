package keepers

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestQuery(t *testing.T) {
	plugin := &keepers{
		logger: log.New(io.Discard, "", 0),
	}
	b, err := plugin.Query(context.Background(), types.ReportTimestamp{})

	assert.NoError(t, err)
	assert.Equal(t, types.Query{}, b)
}

func BenchmarkQuery(b *testing.B) {
	plugin := &keepers{
		logger: log.New(io.Discard, "", 0),
	}

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
			ExpectedObservation: nil,
			ExpectedErr:         fmt.Errorf("test error: failed to sample upkeeps for observation"),
		},
		{
			Name: "Filter to Empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*ktypes.UpkeepResult{
				{Key: ktypes.UpkeepKey([]byte("1|1")), State: ktypes.NotEligible},
				{Key: ktypes.UpkeepKey([]byte("1|2")), State: ktypes.NotEligible},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{})),
		},
		{
			Name: "Filter to Non-empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*ktypes.UpkeepResult{
				{Key: ktypes.UpkeepKey([]byte("1|1")), State: ktypes.NotEligible},
				{Key: ktypes.UpkeepKey([]byte("1|2")), State: ktypes.Eligible},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{[]byte("1|2")})),
		},
		{
			Name: "Reduce Key List to Observation Limit",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: []*ktypes.UpkeepResult{
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000001")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000002")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000003")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000004")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000005")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000006")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000007")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000008")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000009")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000010")), State: ktypes.Eligible},
				{Key: ktypes.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000011")), State: ktypes.Eligible},
			},
			ExpectedObservation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000001"),
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000002"),
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000003"),
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000004"),
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000005"),
				[]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000006"),
			})),
		},
	}

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			ms := new(MockedUpkeepService)
			mf := new(MockedFilterer)

			plugin := &keepers{
				service: ms,
				logger:  log.New(io.Discard, "", 0),
				filter:  mf,
			}

			mf.Mock.On("Filter").Return(func(k ktypes.UpkeepKey) bool {
				return true
			})

			ctx, cancel := test.Ctx()
			ms.Mock.On("SampleUpkeeps", mock.Anything).Return(test.SampleSet, test.SampleErr)

			b, err := plugin.Observation(ctx, types.ReportTimestamp{}, types.Query{})
			cancel()

			if test.ExpectedErr == nil {
				assert.NoError(t, err, "no error expected for test %d; got %s", i+1, err)
			} else {
				assert.Contains(t, err.Error(), test.ExpectedErr.Error(), "error should match expected for test %d", i+1)
			}

			assert.Equal(t, test.ExpectedObservation, b, "observation mismatch for test %d", i+1)
			assert.LessOrEqual(t, len(b), 1000, "observation length should be less than expected")

			// assert that the context passed to Observation is also passed to the service
			ms.Mock.AssertExpectations(t)
		})
	}
}

func BenchmarkObservation(b *testing.B) {
	ms := new(MockedUpkeepService)
	mf := &BenchmarkMockedFilterer{}

	plugin := &keepers{
		service: ms,
		logger:  log.New(io.Discard, "", 0),
		filter:  mf,
	}

	set := make([]*ktypes.UpkeepResult, 2, 100)
	set[0] = &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte("1|1")), State: ktypes.Eligible}
	set[1] = &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte("1|2")), State: ktypes.Eligible}

	for i := 2; i < 100; i++ {
		set = append(set, &ktypes.UpkeepResult{Key: ktypes.UpkeepKey([]byte(fmt.Sprintf("1|%d", i+1))), State: ktypes.NotEligible})
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
		Description  string
		Ctx          func() (context.Context, func())
		Observations []types.AttributedObservation
		FilterOut    []ktypes.UpkeepKey
		Checks       []struct {
			K []ktypes.UpkeepKey
			R ktypes.UpkeepResults
			E error
		}
		Perform        []int
		EncodeErr      error
		ExpectedReport []byte
		ExpectedBool   bool
		ExpectedErr    error
	}{
		{
			Name:        "Single Common Upkeep",
			Description: "A single report should be created when all observations match",
			Ctx:         func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1"))}))},
			},
			Checks: []struct {
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
				},
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
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
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
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{}},
					E: ErrMockTestError,
				},
			},
			ExpectedBool: false,
			ExpectedErr:  ErrMockTestError,
		},
		{
			Name:        "Unsorted Observations",
			Description: "A single report should be created from the id of the earliest block number and sorting of keys should take place in this step",
			Ctx:         func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey("1|2"), ktypes.UpkeepKey("1|1")}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")}))},
			},
			Checks: []struct {
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
				},
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
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name:        "Skip Already Performed",
			Description: "Observations that had already been added to the filter should not be checked nor should they be performed",
			Ctx:         func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|1")), ktypes.UpkeepKey([]byte("1|2"))}))},
				{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{ktypes.UpkeepKey([]byte("1|2"))}))},
			},
			FilterOut: []ktypes.UpkeepKey{
				ktypes.UpkeepKey([]byte("1|1")),
			},
			Checks: []struct {
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|2")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 2, []byte("abcd"))),
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
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.NotEligible, PerformData: []byte("abcd")}},
				},
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|2")},
					R: ktypes.UpkeepResults{{State: ktypes.NotEligible, PerformData: []byte("abcd")}},
				},
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
				K []ktypes.UpkeepKey
				R ktypes.UpkeepResults
				E error
			}{
				{
					K: []ktypes.UpkeepKey{ktypes.UpkeepKey("1|1")},
					R: ktypes.UpkeepResults{{State: ktypes.Eligible, PerformData: []byte("abcd")}},
				},
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
			mf := new(MockedFilterer)

			plugin := &keepers{
				service: ms,
				encoder: me,
				logger:  log.New(io.Discard, "", 0),
				filter:  mf,
			}
			ctx, cancel := test.Ctx()

			mf.Mock.On("Filter").Return(func(k ktypes.UpkeepKey) bool {
				for _, key := range test.FilterOut {
					if string(k) == string(key) {
						return false
					}
				}
				return true
			})

			// set up upkeep checks with the mocked service
			for _, check := range test.Checks {
				for i, k := range check.K {
					check.R[i].Key = k
				}
				fmt.Println("check.K", check.K)
				ms.Mock.On("CheckUpkeep", mock.Anything, check.K).Return(check.R, check.E)
			}

			if len(test.Perform) > 0 {
				toPerform := make([]ktypes.UpkeepResult, len(test.Perform))
				for i, p := range test.Perform {
					u := test.Checks[p]
					for j, k := range u.K {
						u.R[j].Key = k
						toPerform[i] = u.R[j]
					}
					mf.Mock.On("Add", u.K).Return(nil)
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

			assert.LessOrEqual(t, len(r), 10_000, "report length should be less than limit")

			ms.Mock.AssertExpectations(t)
			me.Mock.AssertExpectations(t)
		})
	}
}

func BenchmarkReport(b *testing.B) {
	ms := &BenchmarkMockUpkeepService{}
	me := &BenchmarkMockedReportEncoder{}
	mf := &BenchmarkMockedFilterer{}

	plugin := &keepers{
		service: ms,
		encoder: me,
		logger:  log.New(io.Discard, "", 0),
		filter:  mf,
	}

	key1 := ktypes.UpkeepKey([]byte("1|1"))
	key2 := ktypes.UpkeepKey([]byte("1|2"))
	key3 := ktypes.UpkeepKey([]byte("2|1"))
	data := []byte("abcd")

	encoded := mustEncodeKeys([]ktypes.UpkeepKey{key1, key2})

	set := ktypes.UpkeepResults{
		{Key: key1, State: ktypes.Eligible, PerformData: data},
		{Key: key2, State: ktypes.Eligible, PerformData: data},
	}
	observations := []types.AttributedObservation{
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{key2, key3}))},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
	}

	ms.rtnCheck = ktypes.UpkeepResults{set[0]}

	for i := 1; i <= 4; i++ {
		ob := observations[0 : i*4]

		b.Run(fmt.Sprintf("%d Nodes", len(ob)), func(b *testing.B) {
			b.ResetTimer()

			// run the Observation function b.N times
			for n := 0; n < b.N; n++ {
				ctx := context.Background()
				me.rtnBytes = []byte(fmt.Sprintf("%d+%s", 1, data))

				b.StartTimer()
				_, _, err := plugin.Report(ctx, types.ReportTimestamp{}, types.Query{}, ob)
				b.StopTimer()

				if err != nil {
					b.Fail()
				}
			}
		})
	}
}

func TestShouldAcceptFinalizedReport(t *testing.T) {
	plugin := &keepers{
		logger: log.New(io.Discard, "", 0),
	}
	ok, err := plugin.ShouldAcceptFinalizedReport(context.Background(), types.ReportTimestamp{}, types.Report{})

	assert.Equal(t, false, ok)
	assert.NoError(t, err)
}

func BenchmarkShouldAcceptFinalizedReport(b *testing.B) {
	me := &BenchmarkMockedReportEncoder{}
	mf := &BenchmarkMockedFilterer{}

	plugin := &keepers{
		logger:  log.New(io.Discard, "", 0),
		encoder: me,
		filter:  mf,
	}

	// run the ShouldAcceptFinalizedReport function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.ShouldAcceptFinalizedReport(context.Background(), types.ReportTimestamp{}, types.Report{})
		if err != nil {
			b.Fail()
		}
	}
}

func TestShouldTransmitAcceptedReport(t *testing.T) {
	tests := []struct {
		Name           string
		Description    string
		ReportContents []ktypes.UpkeepResult
		TransmitChecks []struct {
			K ktypes.UpkeepKey
			B bool
		}
		DecodeErr    error
		ExpectedBool bool
		Err          error
	}{
		{
			Name:           "Decode Error",
			Description:    "Should transmit should return false and a wrapped error when report decoding fails",
			ReportContents: nil,
			DecodeErr:      fmt.Errorf("wrapped error"),
			ExpectedBool:   false,
			Err:            fmt.Errorf("failed to get ids from report"),
		},
		{
			Name:           "Empty Report",
			Description:    "Should transmit should return false with an error when report does not contain items",
			ReportContents: nil,
			ExpectedBool:   false,
			Err:            fmt.Errorf("no ids in report"),
		},
		{
			Name:        "Is Transmitted",
			Description: "Should not transmit if call to filter transmitted returns true",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: ktypes.UpkeepKey("1|1"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: ktypes.UpkeepKey("1|1"), B: true},
			},
			ExpectedBool: false,
			Err:          nil,
		},
		{
			Name:        "Is Not Transmitted",
			Description: "Should transmit if call to filter transmitted returns false",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: ktypes.UpkeepKey("1|1"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: ktypes.UpkeepKey("1|1"), B: false},
			},
			ExpectedBool: true,
			Err:          nil,
		},
		{
			Name:        "Multiple Results w/ Last Transmitted",
			Description: "Should not transmit if one key in report has been transmitted",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: ktypes.UpkeepKey("1|1"),
				},
				{
					Key: ktypes.UpkeepKey("1|2"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: ktypes.UpkeepKey("1|1"), B: false},
				{K: ktypes.UpkeepKey("1|2"), B: true},
			},
			ExpectedBool: false,
			Err:          nil,
		},
	}

	for _, test := range tests {
		ms := new(MockedUpkeepService)
		me := new(MockedReportEncoder)
		mf := new(MockedFilterer)

		plugin := &keepers{
			logger:  log.New(io.Discard, "", 0),
			encoder: me,
			filter:  mf,
			service: ms,
		}

		me.Mock.On("DecodeReport", []byte("abc")).Return(test.ReportContents, test.DecodeErr)

		// set the transmit filters before calling the function in test
		for _, a := range test.TransmitChecks {
			mf.Mock.On("IsTransmissionConfirmed", a.K).Return(a.B)
		}

		ctx := context.Background()
		ok, err := plugin.ShouldTransmitAcceptedReport(ctx, types.ReportTimestamp{Epoch: 5, Round: 2}, types.Report([]byte("abc")))

		assert.Equal(t, test.ExpectedBool, ok)
		if test.Err != nil {
			assert.Contains(t, err.Error(), test.Err.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}

func BenchmarkShouldTransmitAcceptedReport(b *testing.B) {
	me := &BenchmarkMockedReportEncoder{}
	mf := &BenchmarkMockedFilterer{}

	me.rtnKeys = []ktypes.UpkeepResult{
		{Key: ktypes.UpkeepKey([]byte("1|1"))},
	}

	plugin := &keepers{
		logger:  log.New(io.Discard, "", 0),
		encoder: me,
		filter:  mf,
	}

	// run the ShouldTransmitAcceptedReport function b.N times
	for n := 0; n < b.N; n++ {
		_, err := plugin.ShouldTransmitAcceptedReport(context.Background(), types.ReportTimestamp{}, types.Report{})
		if err != nil {
			b.Logf("error encountered during benchmark: %s", err)
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

func (_m *MockedUpkeepService) SampleUpkeeps(ctx context.Context, filters ...func(ktypes.UpkeepKey) bool) ([]*ktypes.UpkeepResult, error) {
	arguments := []interface{}{ctx}
	if len(filters) > 0 {
		args := make([]interface{}, len(arguments)+len(filters))
		args[0] = arguments[0]

		for i, filter := range filters {
			args[i+1] = filter
		}

		copy(arguments, args)
	}

	ret := _m.Mock.Called(arguments...)

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

func (_m *MockedUpkeepService) CheckUpkeep(ctx context.Context, keys ...ktypes.UpkeepKey) (ktypes.UpkeepResults, error) {
	ret := _m.Mock.Called(ctx, keys)

	var r0 ktypes.UpkeepResults
	if rf, ok := ret.Get(0).(func() ktypes.UpkeepResults); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ktypes.UpkeepResults)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockedUpkeepService) LockoutUpkeep(ctx context.Context, key ktypes.UpkeepIdentifier) error {
	return _m.Mock.Called(ctx, key).Error(0)
}

func (_m *MockedUpkeepService) IsUpkeepLocked(ctx context.Context, key ktypes.UpkeepIdentifier) (bool, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	return r0, ret.Error(1)
}

type BenchmarkMockUpkeepService struct {
	rtnCheck ktypes.UpkeepResults
}

func (_m *BenchmarkMockUpkeepService) SampleUpkeeps(ctx context.Context, filters ...func(ktypes.UpkeepKey) bool) ([]*ktypes.UpkeepResult, error) {
	return nil, nil
}

func (_m *BenchmarkMockUpkeepService) CheckUpkeep(ctx context.Context, keys ...ktypes.UpkeepKey) (ktypes.UpkeepResults, error) {
	return _m.rtnCheck, nil
}

func (_m *BenchmarkMockUpkeepService) LockoutUpkeep(ctx context.Context, key ktypes.UpkeepIdentifier) error {
	return nil
}

func (_m *BenchmarkMockUpkeepService) IsUpkeepLocked(ctx context.Context, key ktypes.UpkeepIdentifier) (bool, error) {
	return false, nil
}

func mustEncodeKeys(keys []ktypes.UpkeepKey) []byte {
	b, _ := encode(keys)
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

func (_m *MockedReportEncoder) DecodeReport(report []byte) ([]ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(report)

	var r0 []ktypes.UpkeepResult
	if rf, ok := ret.Get(0).(func() []ktypes.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ktypes.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}

type MockedFilterer struct {
	mock.Mock
}

func (_m *MockedFilterer) Filter() func(ktypes.UpkeepKey) bool {
	ret := _m.Mock.Called()

	var r0 func(ktypes.UpkeepKey) bool
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(func(ktypes.UpkeepKey) bool)
	}

	return r0
}

func (_m *MockedFilterer) Accept(key ktypes.UpkeepKey) error {
	return _m.Mock.Called(key).Error(0)
}

func (_m *MockedFilterer) IsTransmissionConfirmed(key ktypes.UpkeepKey) bool {
	ret := _m.Mock.Called(key)

	var r0 bool
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type MockedRandomSource struct {
	mock.Mock
}

func (_m *MockedRandomSource) Int63() int64 {
	ret := _m.Mock.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(int64)
		}
	}

	return r0
}

func (_m *MockedRandomSource) Seed(seed int64) {
	_m.Mock.Called(seed)
}

type BenchmarkMockedReportEncoder struct {
	rtnBytes []byte
	rtnKeys  []ktypes.UpkeepResult
}

func (_m *BenchmarkMockedReportEncoder) EncodeReport(toReport []ktypes.UpkeepResult) ([]byte, error) {
	return _m.rtnBytes, nil
}

func (_m *BenchmarkMockedReportEncoder) DecodeReport(report []byte) ([]ktypes.UpkeepResult, error) {
	return _m.rtnKeys, nil
}

type BenchmarkMockedRegistry struct {
	rtnKeys []ktypes.UpkeepKey
	rtnRes  ktypes.UpkeepResult
	rtnId   ktypes.UpkeepIdentifier
}

func (_m *BenchmarkMockedRegistry) GetActiveUpkeepKeys(ctx context.Context, key ktypes.BlockKey) ([]ktypes.UpkeepKey, error) {
	return _m.rtnKeys, nil
}

func (_m *BenchmarkMockedRegistry) CheckUpkeep(ctx context.Context, keys ...ktypes.UpkeepKey) (bool, ktypes.UpkeepResult, error) {
	return true, _m.rtnRes, nil
}

func (_m *BenchmarkMockedRegistry) IdentifierFromKey(key ktypes.UpkeepKey) (ktypes.UpkeepIdentifier, error) {
	return _m.rtnId, nil
}

type BenchmarkMockedFilterer struct{}

func (_m *BenchmarkMockedFilterer) Filter() func(ktypes.UpkeepKey) bool {
	return func(ktypes.UpkeepKey) bool {
		return true
	}
}

func (_m *BenchmarkMockedFilterer) Accept(key ktypes.UpkeepKey) error {
	return nil
}

func (_m *BenchmarkMockedFilterer) IsTransmissionConfirmed(key ktypes.UpkeepKey) bool {
	return false
}
