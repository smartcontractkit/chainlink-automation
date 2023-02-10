package keepers

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

type testStringer string

func (ts testStringer) String() string {
	return string(ts)
}

func TestCryptoShuffler(t *testing.T) {
	expectected := []int{1, 2, 3, 4, 5}
	test := make([]int, len(expectected))
	copy(test, expectected)

	sh := &cryptoShuffler[int]{}
	sh.Shuffle(test)

	assert.NotEqual(t, test, expectected)
	for _, value := range expectected {
		assert.Contains(t, test, value)
	}
}

func TestDedupe(t *testing.T) {
	tests := []struct {
		Name           string
		Sets           [][]testStringer
		ExpectedResult []testStringer
		ExpectedError  error
	}{
		{Name: "Empty Matching Sets", ExpectedResult: nil, ExpectedError: ErrNotEnoughInputs},
		{Name: "Single Matching Set", Sets: [][]testStringer{{"1", "2", "3"}}, ExpectedResult: []testStringer{"1", "2", "3"}},
		{Name: "Double Identical", Sets: [][]testStringer{{"1", "2", "3"}, {"1", "2", "3"}}, ExpectedResult: []testStringer{"1", "2", "3"}},
		{Name: "Double No Match", Sets: [][]testStringer{{"1", "2", "3"}, {"4", "5", "6"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5", "6"}},
		{Name: "Triple Identical", Sets: [][]testStringer{{"1", "2", "3"}, {"1", "2", "3"}, {"1", "2", "3"}}, ExpectedResult: []testStringer{"1", "2", "3"}},
		{Name: "Triple No Match", Sets: [][]testStringer{{"1", "2", "3"}, {"4", "5", "6"}, {"7", "8", "9"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
		{Name: "Double w/ Single Overlap", Sets: [][]testStringer{{"1", "2", "3"}, {"3", "4", "5"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5"}},
		{Name: "Double w/ Single Overlap and Gap", Sets: [][]testStringer{{"1", "2", "3", "6", "7"}, {"3", "4", "5"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5", "6", "7"}},
		{Name: "Double w/ Double Overlap and Gap", Sets: [][]testStringer{{"1", "2", "3", "4", "6", "7"}, {"3", "4", "5"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5", "6", "7"}},
		{Name: "Triple w/ Single Overlap", Sets: [][]testStringer{{"1", "2", "3"}, {"1", "2", "3"}, {"3", "4", "5"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5"}},
		{Name: "Triple w/ Single Overlap and Gap", Sets: [][]testStringer{{"1", "2", "4"}, {"1", "2", "3", "4"}, {"3", "4", "5"}}, ExpectedResult: []testStringer{"1", "2", "3", "4", "5"}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var matches []testStringer
			var err error

			matches, err = dedupe(test.Sets)
			if test.ExpectedError != nil {
				assert.ErrorIs(t, err, test.ExpectedError)
			} else {
				sort.Slice(matches, func(i, j int) bool {
					return matches[i] < matches[j]
				})
				assert.Equal(t, test.ExpectedResult, matches)
			}
		})
	}
}

func TestShuffledDedupedKeyList(t *testing.T) {
	for _, tc := range []struct {
		observations [][]ktypes.UpkeepKey
		expected     []ktypes.UpkeepKey
		wantErr      bool
	}{
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("1|1"),
					chain.UpkeepKey("2|1"),
				},
				{
					chain.UpkeepKey("3|1"),
					chain.UpkeepKey("1|2"),
				},
				{
					chain.UpkeepKey("1|2"),
					chain.UpkeepKey("1|3"),
				},
				{
					chain.UpkeepKey("1|3"),
					chain.UpkeepKey("2|3"),
				},
				{
					chain.UpkeepKey("1|1"),
					chain.UpkeepKey("2|1"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("1|3"),
				chain.UpkeepKey("1|1"),
				chain.UpkeepKey("1|2"),
			},
		},
		{
			observations: nil,
			expected:     nil,
			wantErr:      true,
		},
		{
			observations: [][]ktypes.UpkeepKey{},
			expected:     nil,
			wantErr:      true,
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey(""),
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("2|1"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("2|1"),
			},
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("2|1"),
					chain.UpkeepKey("2|3"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("2|3"),
				chain.UpkeepKey("2|1"),
			},
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("2|1"),
					chain.UpkeepKey("2|5"),
					chain.UpkeepKey("2|3"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("2|5"),
				chain.UpkeepKey("2|1"),
				chain.UpkeepKey("2|3"),
			},
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("1|1"),
					chain.UpkeepKey("2|1"),
					chain.UpkeepKey("3|2"),
					chain.UpkeepKey("2|3"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("2|2"),
				chain.UpkeepKey("2|1"),
				chain.UpkeepKey("2|3"),
			},
		},
		{
			observations: [][]ktypes.UpkeepKey{
				{
					chain.UpkeepKey("2|1"),
				},
				{
					chain.UpkeepKey("2|5"),
				},
				{
					chain.UpkeepKey("2|3"),
				},
			},
			expected: []ktypes.UpkeepKey{
				chain.UpkeepKey("2|3"),
				chain.UpkeepKey("2|1"),
				chain.UpkeepKey("2|5"),
			},
		},
	} {
		var attr []types.AttributedObservation
		for _, o := range tc.observations {
			b, err := limitedLengthEncode(o, maxObservationLength)
			assert.NoError(t, err)

			attr = append(attr, types.AttributedObservation{
				Observation: b,
			})
		}

		var key [16]byte

		filterFn := func(ktypes.UpkeepKey) bool {
			return true
		}

		result, err := shuffledDedupedKeyList(attr, key, filterFn)
		assert.Equal(t, tc.expected, result)

		if tc.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestSortedDedup_Error(t *testing.T) {
	obs := []types.AttributedObservation{{Observation: types.Observation([]byte("incorrectly encoded"))}}
	_, err := shuffledDedupedKeyList(obs, [16]byte{})
	assert.NotNil(t, err)
}

func BenchmarkSortedDedupedKeyListFunc(b *testing.B) {
	key1 := chain.UpkeepKey([]byte("1|1"))
	key2 := chain.UpkeepKey([]byte("1|2"))
	key3 := chain.UpkeepKey([]byte("2|1"))

	encoded := mustEncodeKeys([]ktypes.UpkeepKey{key1, key2})

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

	for i := 1; i <= 4; i++ {
		ob := observations[0 : i*4]

		b.Run(fmt.Sprintf("%d Nodes", len(ob)), func(b *testing.B) {
			b.ResetTimer()

			// run the Observation function b.N times
			for n := 0; n < b.N; n++ {

				b.StartTimer()
				_, err := shuffledDedupedKeyList(ob, [16]byte{})
				b.StopTimer()

				if err != nil {
					b.Fail()
				}
			}
		})
	}
}

func TestSampleFromProbability(t *testing.T) {
	tests := []struct {
		Name         string
		Rounds       int
		Nodes        int
		Probability  float32
		ExpectedErr  error
		ExpectedHigh float32
		ExpectedLow  float32
	}{
		{Name: "Negative Rounds", Rounds: -1, ExpectedErr: fmt.Errorf("number of rounds must be greater than 0")},
		{Name: "Zero Rounds", ExpectedErr: fmt.Errorf("number of rounds must be greater than 0")},
		{Name: "Negative Nodes", Rounds: 1, Nodes: -1, ExpectedErr: fmt.Errorf("number of nodes must be greater than 0")},
		{Name: "Zero Nodes", Rounds: 1, ExpectedErr: fmt.Errorf("number of nodes must be greater than 0")},
		{Name: "Probability Greater Than 1", Rounds: 1, Nodes: 1, Probability: 2, ExpectedErr: fmt.Errorf("probability must be less than 1 and greater than 0")},
		{Name: "Probability Less Than 0", Rounds: 1, Nodes: 1, Probability: -1, ExpectedErr: fmt.Errorf("probability must be less than 1 and greater than 0")},
		{Name: "Probability Equal to 0", Rounds: 1, Nodes: 1, Probability: 0, ExpectedErr: fmt.Errorf("probability must be less than 1 and greater than 0")},
		{Name: "Valid", Rounds: 1, Nodes: 4, Probability: 0.975, ExpectedErr: nil, ExpectedHigh: 0.61, ExpectedLow: 0.59},
	}

	for _, test := range tests {
		v, err := sampleFromProbability(test.Rounds, test.Nodes, test.Probability)
		assert.Equal(t, test.ExpectedErr, err)

		if test.ExpectedHigh == 0.0 {
			assert.Equal(t, float32(v), test.ExpectedHigh)
		} else {
			assert.Greater(t, float32(v), test.ExpectedLow)
			assert.Less(t, float32(v), test.ExpectedHigh)
		}
	}
}

func TestLowest(t *testing.T) {

	values := []int64{-3, -1, 0, 1, 3}
	var expected int64 = -3

	result := lowest(values)

	assert.Equal(t, expected, result)
}

func TestLimitedLengthEncode(t *testing.T) {
	tests := []struct {
		Name      string
		KeyLength int
		KeyCount  int
		MaxLength int
	}{
		{Name: "Few Short Keys", KeyLength: 14, KeyCount: 10, MaxLength: 1000},
		{Name: "Many Short Keys", KeyLength: 4, KeyCount: 100, MaxLength: 1000},
		{Name: "Few Long Keys", KeyLength: 40, KeyCount: 10, MaxLength: 1000},
		{Name: "Many Long Keys", KeyLength: 62, KeyCount: 100, MaxLength: 1000},
	}

	for _, test := range tests {
		keys := make([]ktypes.UpkeepKey, test.KeyCount)
		for i := 0; i < test.KeyCount; i++ {
			byt := make([]byte, test.KeyLength)
			keys[i] = chain.UpkeepKey(byt)
		}

		b, err := limitedLengthEncode(keys, test.MaxLength)
		t.Logf("length: %d", len(b))

		assert.NoError(t, err)
		assert.LessOrEqual(t, len(b), test.MaxLength)
	}
}

func FuzzLimitedLengthEncode(f *testing.F) {
	f.Add(4, 10)
	f.Fuzz(func(t *testing.T, a int, b int) {
		// only accept fuzz values of key length between 2 and 700 and number
		// of keys greater than or equal to 0.
		// the number 700 was chosen because a key length larger than this
		// should be paired with a higher limit anyway. this test is scoped to
		// a single encoded limit while fuzzing the keys and key lengths.
		// keys are randomized in length to ensure outcome remains within limits
		if a < 0 || b <= 1 || b >= 700 {
			return
		}

		keys := make([]ktypes.UpkeepKey, a)
		for i := 0; i < a; i++ {
			k := strings.Repeat("1", rand.Intn(b)+3)
			k = k[:1] + "|" + k[2:]
			keys[i] = chain.UpkeepKey(k)
		}

		bt, err := limitedLengthEncode(keys, 1000)

		assert.NoError(t, err)
		assert.LessOrEqual(t, len(bt), 1000, "keys: %d; length: %d", a, b)

		if a > 0 {
			output := make([]ktypes.UpkeepKey, 0)
			outputKeys := make([]chain.UpkeepKey, 0)
			err = decode(bt, &outputKeys)
			assert.NoError(t, err)

			for _, o := range outputKeys {
				output = append(output, o)
			}

			assert.Greater(t, len(bt), 0, "length of bytes :: keys: %d; length: %d", a, b)
			assert.Greater(t, len(output), 0, "min number of keys :: keys: %d; length: %d", a, b)
			assert.LessOrEqual(t, len(output), a, "max number of keys :: keys: %d; length: %d", a, b)
		}
	})
}
