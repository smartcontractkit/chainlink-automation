package keepers

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	bigBlockKey, _ := big.NewInt(0).SetString("100000000000100000000000100000000000100000000000100000000001", 10)
	tests := []struct {
		Name                string
		Ctx                 func() (context.Context, func())
		SampleSet           ktypes.UpkeepResults
		LatestBlock         *big.Int
		ServiceError        bool
		SampleErr           error
		ExpectedObservation types.Observation
		ExpectedErr         error
	}{
		{
			Name:        "Empty Set",
			Ctx:         func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:   ktypes.UpkeepResults{},
			LatestBlock: big.NewInt(1),
			ExpectedObservation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{
				BlockKey:          ktypes.BlockKey("1"),
				UpkeepIdentifiers: []ktypes.UpkeepIdentifier{},
			})),
		},
		{
			Name:        "Timer Context",
			Ctx:         func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			SampleSet:   ktypes.UpkeepResults{},
			LatestBlock: big.NewInt(2),
			ExpectedObservation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{
				BlockKey:          ktypes.BlockKey("2"),
				UpkeepIdentifiers: []ktypes.UpkeepIdentifier{},
			}))},
		{
			Name:                "Upkeep Service Error",
			Ctx:                 func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet:           ktypes.UpkeepResults{},
			LatestBlock:         big.NewInt(3),
			SampleErr:           fmt.Errorf("test error"),
			ExpectedObservation: nil,
			ServiceError:        true,
			ExpectedErr:         fmt.Errorf("test error: failed to sample upkeeps for observation"),
		},
		{
			Name: "Filter to Empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: ktypes.UpkeepResults{
				{Key: chain.UpkeepKey("1|1"), State: ktypes.NotEligible},
				{Key: chain.UpkeepKey("1|2"), State: ktypes.NotEligible},
			},
			LatestBlock: big.NewInt(1),
			ExpectedObservation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{
				BlockKey:          ktypes.BlockKey("1"),
				UpkeepIdentifiers: []ktypes.UpkeepIdentifier{},
			})),
		},
		{
			Name: "Filter to Non-empty Set",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: ktypes.UpkeepResults{
				{Key: chain.UpkeepKey([]byte("1|1")), State: ktypes.NotEligible},
				{Key: chain.UpkeepKey([]byte("1|2")), State: ktypes.Eligible},
			},
			LatestBlock: big.NewInt(1),
			ExpectedObservation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{
				BlockKey: "1",
				UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
					ktypes.UpkeepIdentifier("2"),
				},
			})),
		},
		{
			Name: "Reduce Key List to Observation Limit",
			Ctx:  func() (context.Context, func()) { return context.Background(), func() {} },
			SampleSet: ktypes.UpkeepResults{
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000001")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000002")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000003")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000004")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000005")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000006")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000007")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000008")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000009")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000010")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000011")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000012")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000013")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000014")), State: ktypes.Eligible},
				{Key: chain.UpkeepKey([]byte("100000000000100000000000100000000000100000000000100000000001|100000000000100000000000100000000000100000000015")), State: ktypes.Eligible},
			},
			LatestBlock: bigBlockKey,
			ExpectedObservation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{
				BlockKey: "100000000000100000000000100000000000100000000000100000000001",
				UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000001"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000002"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000003"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000004"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000005"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000006"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000007"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000008"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000009"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000010"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000011"),
					ktypes.UpkeepIdentifier("100000000000100000000000100000000000100000000012"),
				},
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
			if !test.ServiceError {
				ms.Mock.On("LatestBlock", mock.Anything).Return(test.LatestBlock, nil)
			}

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

	set := make(ktypes.UpkeepResults, 2, 100)
	set[0] = ktypes.UpkeepResult{Key: chain.UpkeepKey("1|1"), State: ktypes.Eligible}
	set[1] = ktypes.UpkeepResult{Key: chain.UpkeepKey("1|2"), State: ktypes.Eligible}

	for i := 2; i < 100; i++ {
		set = append(set, ktypes.UpkeepResult{Key: chain.UpkeepKey(fmt.Sprintf("1|%d", i+1)), State: ktypes.NotEligible})
	}

	b.ResetTimer()
	// run the Observation function b.N times
	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		ms.Mock.On("SampleUpkeeps", mock.Anything).Return(set, nil)

		b.StartTimer()
		_, err := plugin.Observation(ctx, types.ReportTimestamp{}, types.Query{})
		b.StopTimer()

		if err != nil {
			b.Fail()
		}
	}
}

func TestReport(t *testing.T) {
	type checks struct {
		K []ktypes.UpkeepKey
		R ktypes.UpkeepResults
		E error
	}

	tests := []struct {
		Name           string
		Description    string
		ReportGasLimit uint32
		Ctx            func() (context.Context, func())
		Observations   []types.AttributedObservation
		FilterOut      []ktypes.UpkeepKey
		Checks         checks
		Perform        []int
		EncodeErr      error
		ExpectedReport []byte
		ExpectedBool   bool
		ExpectedErr    error
	}{
		{
			Name:           "Single Common Upkeep",
			Description:    "A single report should be created when all observations match",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|1"),
				},
				R: ktypes.UpkeepResults{
					{Key: chain.UpkeepKey("1|1"), State: ktypes.Eligible, PerformData: []byte("abcd")},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name:           "Forward Context",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|1"),
				},
				R: ktypes.UpkeepResults{
					{Key: chain.UpkeepKey("1|1"), State: ktypes.Eligible, PerformData: []byte("abcd")},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name:           "Wrap Error",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.WithTimeout(context.Background(), time.Second) },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{chain.UpkeepKey("1|1")},
				R: ktypes.UpkeepResults{{}},
				E: ErrMockTestError,
			},
			ExpectedBool: false,
			ExpectedErr:  ErrMockTestError,
		},
		{
			Name:           "Unsorted Observations",
			Description:    "A single report should be created from the id of the earliest block number and sorting of keys should take place in this step",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("2"),
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|1"),
					chain.UpkeepKey("1|2"),
				},
				R: ktypes.UpkeepResults{
					{
						Key:         chain.UpkeepKey("1|1"),
						State:       ktypes.Eligible,
						PerformData: []byte("abcd"),
					},
					{
						Key:         chain.UpkeepKey("1|2"),
						State:       ktypes.NotEligible,
						PerformData: []byte("abcd"),
					},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 1, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name:           "Skip Already Performed",
			Description:    "Observations that had already been added to the filter should not be checked nor should they be performed",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
			},
			FilterOut: []ktypes.UpkeepKey{
				chain.UpkeepKey("1|1"),
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|2"),
				},
				R: ktypes.UpkeepResults{
					{Key: chain.UpkeepKey("1|2"), State: ktypes.Eligible, PerformData: []byte("abcd")},
				},
			},
			Perform:        []int{0},
			ExpectedReport: []byte(fmt.Sprintf("%d+%s", 2, []byte("abcd"))),
			ExpectedBool:   true,
		},
		{
			Name:           "Nothing to Report",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("2"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|2"),
					chain.UpkeepKey("1|1"),
				},
				R: ktypes.UpkeepResults{
					{Key: chain.UpkeepKey("1|2"), State: ktypes.NotEligible, PerformData: []byte("abcd")},
					{Key: chain.UpkeepKey("1|1"), State: ktypes.NotEligible, PerformData: []byte("abcd")},
				},
			},
			ExpectedBool: false,
		},
		{
			Name:           "Empty Observations",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{}))},
				{Observation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{}))},
				{Observation: types.Observation(mustEncodeUpkeepObservation(&ktypes.UpkeepObservation{}))},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{},
				R: ktypes.UpkeepResults{},
			},
			ExpectedBool: false,
		},
		{
			Name:           "No Observations",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations:   []types.AttributedObservation{},
			ExpectedBool:   false,
			ExpectedErr:    ErrNotEnoughInputs,
		},
		{
			Name:           "Encoding Error",
			ReportGasLimit: 10000000,
			Ctx:            func() (context.Context, func()) { return context.Background(), func() {} },
			Observations: []types.AttributedObservation{
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
				{Observation: types.Observation(mustEncodeUpkeepObservation(
					&ktypes.UpkeepObservation{
						BlockKey: ktypes.BlockKey("1"),
						UpkeepIdentifiers: []ktypes.UpkeepIdentifier{
							ktypes.UpkeepIdentifier("1"),
						},
					})),
				},
			},
			Checks: checks{
				K: []ktypes.UpkeepKey{
					chain.UpkeepKey("1|1"),
				},
				R: ktypes.UpkeepResults{
					{Key: chain.UpkeepKey("1|1"), State: ktypes.Eligible, PerformData: []byte("abcd")},
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
			me := ktypes.NewMockReportEncoder(t)
			mf := new(MockedFilterer)

			plugin := &keepers{
				service:        ms,
				encoder:        me,
				logger:         log.New(io.Discard, "", 0),
				filter:         mf,
				reportGasLimit: test.ReportGasLimit,
			}
			ctx, cancel := test.Ctx()

			mf.Mock.On("Filter").Return(func(k ktypes.UpkeepKey) bool {
				for _, key := range test.FilterOut {
					if k.String() == key.String() {
						return false
					}
				}
				return true
			})

			// set up upkeep checks with the mocked service
			if len(test.Checks.K) > 0 {
				ms.Mock.On("CheckUpkeep", mock.Anything, test.Checks.K).Return(test.Checks.R, test.Checks.E)
			}

			if len(test.Perform) > 0 {
				toPerform := make([]ktypes.UpkeepResult, len(test.Perform))
				for i, p := range test.Perform {
					toPerform[i] = test.Checks.R[p]
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
			mf.Mock.AssertExpectations(t)
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

	key1 := chain.UpkeepKey([]byte("1|1"))
	key2 := chain.UpkeepKey([]byte("1|2"))
	key3 := chain.UpkeepKey([]byte("2|1"))
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
	tests := []struct {
		Name            string
		Description     string
		ReportContents  []ktypes.UpkeepResult
		AlreadyAccepted []struct {
			K ktypes.UpkeepKey
			B bool
		}
		DecodeErr    error
		ExpectedBool bool
		Err          error
	}{
		{
			Name:           "No upkeeps in report",
			Description:    "Should accept should return false and a wrapped error when report is empty",
			ReportContents: nil,
			ExpectedBool:   false,
			Err:            fmt.Errorf("no ids in report"),
		},
		{
			Name:           "Decode Error",
			Description:    "Should accept should return false with an error when report does not decode",
			ReportContents: nil,
			DecodeErr:      fmt.Errorf("wrapped error"),
			ExpectedBool:   false,
			Err:            fmt.Errorf("failed to decode report"),
		},
		{
			Name:        "Already accepted keys",
			Description: "Should not accept report if calls were previously accepted",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: chain.UpkeepKey("1|1"),
				},
			},
			AlreadyAccepted: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: true},
			},
			ExpectedBool: false,
			Err:          fmt.Errorf("failed to accept key"),
		},
		{
			Name:        "Already accepted partial keys",
			Description: "Should not accept report if calls were previously accepted",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: chain.UpkeepKey("1|1"),
				},
				{
					Key: chain.UpkeepKey("1|2"),
				},
			},
			AlreadyAccepted: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: false},
				{K: chain.UpkeepKey("1|2"), B: true},
			},
			ExpectedBool: false,
			Err:          fmt.Errorf("failed to accept key"),
		},
		{
			Name:        "Accept successfully",
			Description: "Should not accept report if calls were previously accepted",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: chain.UpkeepKey("1|1"),
				},
				{
					Key: chain.UpkeepKey("1|2"),
				},
			},
			AlreadyAccepted: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: false},
				{K: chain.UpkeepKey("1|2"), B: false},
			},
			ExpectedBool: true,
		},
	}

	for _, test := range tests {
		ms := new(MockedUpkeepService)
		me := ktypes.NewMockReportEncoder(t)
		mf := new(MockedFilterer)

		plugin := &keepers{
			logger:  log.New(io.Discard, "", 0),
			encoder: me,
			filter:  mf,
			service: ms,
		}

		me.Mock.On("DecodeReport", []byte("abc")).Return(test.ReportContents, test.DecodeErr)

		// set the transmit filters before calling the function in test
		for _, a := range test.AlreadyAccepted {
			mf.Mock.On("CheckAlreadyAccepted", a.K).Return(a.B)
		}

		if test.ExpectedBool {
			// If shouldAccept is successful then Accept will be called on report which needs to be mocked
			for _, a := range test.ReportContents {
				mf.Mock.On("Accept", a.Key).Return(nil)
			}
		}

		ctx := context.Background()
		ok, err := plugin.ShouldAcceptFinalizedReport(ctx, types.ReportTimestamp{Epoch: 5, Round: 2}, []byte("abc"))
		fmt.Println(ok, err)
		assert.Equal(t, test.ExpectedBool, ok)
		if test.Err != nil {
			assert.Contains(t, err.Error(), test.Err.Error())
		} else {
			assert.NoError(t, err)
		}
	}
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
					Key: chain.UpkeepKey("1|1"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: true},
			},
			ExpectedBool: false,
			Err:          nil,
		},
		{
			Name:        "Is Not Transmitted",
			Description: "Should transmit if call to filter transmitted returns false",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: chain.UpkeepKey("1|1"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: false},
			},
			ExpectedBool: true,
			Err:          nil,
		},
		{
			Name:        "Multiple Results w/ Last Transmitted",
			Description: "Should not transmit if one key in report has been transmitted",
			ReportContents: []ktypes.UpkeepResult{
				{
					Key: chain.UpkeepKey("1|1"),
				},
				{
					Key: chain.UpkeepKey("1|2"),
				},
			},
			TransmitChecks: []struct {
				K ktypes.UpkeepKey
				B bool
			}{
				{K: chain.UpkeepKey("1|1"), B: false},
				{K: chain.UpkeepKey("1|2"), B: true},
			},
			ExpectedBool: false,
			Err:          nil,
		},
	}

	for _, test := range tests {
		ms := new(MockedUpkeepService)
		me := ktypes.NewMockReportEncoder(t)
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
		{Key: chain.UpkeepKey([]byte("1|1"))},
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

func (_m *MockedUpkeepService) LatestBlock(ctx context.Context) (*big.Int, error) {
	ret := _m.Called(ctx)

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(context.Context) *big.Int); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockedUpkeepService) SampleUpkeeps(ctx context.Context, filters ...func(ktypes.UpkeepKey) bool) (ktypes.UpkeepResults, error) {
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

func (_m *BenchmarkMockUpkeepService) LatestBlock(ctx context.Context) (*big.Int, error) {
	return nil, nil
}

func (_m *BenchmarkMockUpkeepService) SampleUpkeeps(ctx context.Context, filters ...func(ktypes.UpkeepKey) bool) (ktypes.UpkeepResults, error) {
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

func mustEncodeUpkeepObservation(o *ktypes.UpkeepObservation) []byte {
	b, _ := encode(o)
	return b
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

func (_m *MockedFilterer) CheckAlreadyAccepted(key ktypes.UpkeepKey) bool {
	ret := _m.Mock.Called(key)

	var r0 bool
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(bool)
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

func (_m *BenchmarkMockedFilterer) CheckAlreadyAccepted(key ktypes.UpkeepKey) bool {
	return false
}

func (_m *BenchmarkMockedFilterer) Accept(key ktypes.UpkeepKey) error {
	return nil
}

func (_m *BenchmarkMockedFilterer) IsTransmissionConfirmed(key ktypes.UpkeepKey) bool {
	return false
}
