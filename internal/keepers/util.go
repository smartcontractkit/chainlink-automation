package keepers

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/cmplx"
	rnd "math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

var (
	ErrNotEnoughInputs = fmt.Errorf("not enough inputs")
)

func filterUpkeeps(upkeeps ktypes.UpkeepResults, filter ktypes.UpkeepState) ktypes.UpkeepResults {
	ret := make(ktypes.UpkeepResults, 0, len(upkeeps))

	for _, up := range upkeeps {
		if up.State == filter {
			ret = append(ret, up)
		}
	}

	return ret
}

func keyList(upkeeps ktypes.UpkeepResults) []ktypes.UpkeepKey {
	ret := make([]ktypes.UpkeepKey, len(upkeeps))

	for i, up := range upkeeps {
		ret[i] = up.Key
	}

	sort.Sort(sortUpkeepKeys(ret))

	return ret
}

type shuffler[T any] interface {
	Shuffle([]T) []T
}

type cryptoShuffler[T any] struct{}

func (_ *cryptoShuffler[T]) Shuffle(a []T) []T {
	r := rnd.New(util.NewCryptoRandSource())
	r.Shuffle(len(a), func(i, j int) {
		a[i], a[j] = a[j], a[i]
	})
	return a
}

type sortUpkeepKeys []ktypes.UpkeepKey

func (s sortUpkeepKeys) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

func (s sortUpkeepKeys) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortUpkeepKeys) Len() int {
	return len(s)
}

func dedupe[T fmt.Stringer](inputs [][]T, filters ...func(T) bool) ([]T, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("%w: must provide at least 1", ErrNotEnoughInputs)
	}

	var max int
	for _, input := range inputs {
		max += len(input)
	}

	output := make([]T, 0, max)
	matched := make(map[string]struct{})
	for _, input := range inputs {
		for _, val := range input {
			add := true
			for _, filter := range filters {
				if !filter(val) {
					add = false
					break
				}
			}

			if !add {
				continue
			}

			key := val.String()
			_, ok := matched[key]
			if !ok {
				matched[key] = struct{}{}
				output = append(output, val)
			}
		}
	}

	return output, nil
}

func shuffleUniqueObservations(observations []types.AttributedObservation, key [16]byte, filters ...func(ktypes.UpkeepKey) bool) ([]ktypes.UpkeepKey, error) {
	if len(observations) == 0 {
		return nil, fmt.Errorf("%w: must provide at least 1 observation", ErrNotEnoughInputs)
	}

	upkeepKeys, err := observationsToUpkeepKeys(observations)
	if err != nil {
		return nil, err
	}

	uniqueKeys, err := dedupe(upkeepKeys, filters...)
	if err != nil {
		return nil, err
	}

	uniqueKeys, err = trimLowerBlocks(uniqueKeys)
	if err != nil {
		return nil, err
	}

	rnd.New(util.NewKeyedCryptoRandSource(key)).Shuffle(len(uniqueKeys), func(i, j int) {
		uniqueKeys[i], uniqueKeys[j] = uniqueKeys[j], uniqueKeys[i]
	})

	return uniqueKeys, nil
}

func trimLowerBlocks(uniqueKeys []ktypes.UpkeepKey) ([]ktypes.UpkeepKey, error) {
	idxMap := make(map[string]int)
	out := make([]ktypes.UpkeepKey, 0, len(uniqueKeys))

	for _, uniqueKey := range uniqueKeys {
		blockKey, upkeepID, err := uniqueKey.BlockKeyAndUpkeepID()
		if err != nil {
			return nil, err
		}

		idx, ok := idxMap[string(upkeepID)]
		if !ok {
			idxMap[string(upkeepID)] = len(out)
			out = append(out, uniqueKey)
			continue
		}

		savedBlockKey, _, err := out[idx].BlockKeyAndUpkeepID()
		if err != nil {
			return nil, err
		}

		if string(blockKey) > string(savedBlockKey) {
			out[idx] = uniqueKey
		}
	}

	return out, nil
}

func observationsToUpkeepKeys(observations []types.AttributedObservation) ([][]ktypes.UpkeepKey, error) {
	var parseErrors int

	res := make([][]ktypes.UpkeepKey, len(observations))

	var allBlockKeys []ktypes.BlockKey
	for i, observation := range observations {
		// a single observation returning an error here can void all other
		// good observations. ensure this loop continues on error, but collect
		// them and throw an error if ALL observations fail at this point.
		// TODO we can't rely on this concrete type for decoding/encoding
		var upkeepObservation *ktypes.UpkeepObservation
		if err := decode(observation.Observation, &upkeepObservation); err != nil {
			parseErrors++
			continue
		}

		allBlockKeys = append(allBlockKeys, upkeepObservation.BlockKey)

		// if we have a non-empty list of upkeep identifiers, use the zeroth upkeep identifier
		if len(upkeepObservation.UpkeepIdentifiers) > 0 {
			res[i] = []ktypes.UpkeepKey{
				chain.NewUpkeepKeyFromBlockAndID(upkeepObservation.BlockKey, upkeepObservation.UpkeepIdentifiers[0]),
			}
		}
	}

	if parseErrors == len(observations) {
		return nil, fmt.Errorf("%w: cannot prepare sorted key list; observations not properly encoded", ErrTooManyErrors)
	}

	medianBlock := calculateMedianBlock(allBlockKeys)

	res, err := recreateKeysWithMedianBlock(medianBlock, res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func recreateKeysWithMedianBlock(medianBlock ktypes.BlockKey, upkeepKeyLists [][]ktypes.UpkeepKey) ([][]ktypes.UpkeepKey, error) {
	var res = make([][]ktypes.UpkeepKey, len(upkeepKeyLists))

	for i, upkeepKeys := range upkeepKeyLists {
		var keys []ktypes.UpkeepKey
		for _, upkeepKey := range upkeepKeys {
			_, upkeepID, err := upkeepKey.BlockKeyAndUpkeepID()
			if err != nil {
				return nil, err
			}
			keys = append(keys, chain.NewUpkeepKeyFromBlockAndID(medianBlock, upkeepID))
		}
		res[i] = keys
	}

	return res, nil
}

func calculateMedianBlock(data []ktypes.BlockKey) ktypes.BlockKey {
	var blockKeyInts []*big.Int

	for _, d := range data {
		blockKeyInt := big.NewInt(0)
		blockKeyInt, _ = blockKeyInt.SetString(string(d), 10)
		blockKeyInts = append(blockKeyInts, blockKeyInt)
	}

	sort.Slice(blockKeyInts, func(i, j int) bool {
		return blockKeyInts[i].Cmp(blockKeyInts[j]) < 0
	})

	// this is a crude median calculation; for a list of an odd number of elements, e.g. [10, 20, 30], the center value
	// is chosen as the median. for a list of an even number of elements, a true median calculation would average the
	// two center elements, e.g. [10, 20, 30, 40] = (20 + 30) / 2 = 25, but we want to constrain our median block to
	// one of the block numbers reported, e.g. either 20 or 30. right now we want to choose the higher block number, e.g.
	// 30. for this reason, the logic for selecting the median value from an odd number of elements is the same as the
	// logic for selecting the median value from an even number of elements
	var median *big.Int
	if l := len(blockKeyInts); l == 0 {
		median = big.NewInt(0)
	} else {
		median = blockKeyInts[l/2]
	}

	return ktypes.BlockKey(median.String())
}

func sampleFromProbability(rounds, nodes int, probability float32) (sampleRatio, error) {
	var ratio sampleRatio

	if rounds <= 0 {
		return ratio, fmt.Errorf("number of rounds must be greater than 0")
	}

	if nodes <= 0 {
		return ratio, fmt.Errorf("number of nodes must be greater than 0")
	}

	if probability > 1 || probability <= 0 {
		return ratio, fmt.Errorf("probability must be less than 1 and greater than 0")
	}

	r := complex(float64(rounds), 0)
	n := complex(float64(nodes), 0)
	p := complex(float64(probability), 0)

	g := -1.0 * (p - 1.0)
	x := cmplx.Pow(cmplx.Pow(g, 1.0/r), 1.0/n)
	rat := cmplx.Abs(-1.0 * (x - 1.0))
	rat = math.Round(rat/0.01) * 0.01
	ratio = sampleRatio(float32(rat))

	return ratio, nil
}

func lowest(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	return values[0]
}

type syncedArray[T any] struct {
	data []T
	mu   sync.RWMutex
}

func newSyncedArray[T any]() *syncedArray[T] {
	return &syncedArray[T]{
		data: []T{},
	}
}

func (a *syncedArray[T]) Append(vals ...T) *syncedArray[T] {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.data = append(a.data, vals...)
	return a
}

func (a *syncedArray[T]) Values() []T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

func limitedLengthEncode(obs *ktypes.UpkeepObservation, limit int) ([]byte, error) {
	if len(obs.UpkeepIdentifiers) == 0 {
		return encode(obs)
	}

	var res []byte
	for i := range obs.UpkeepIdentifiers {
		b, err := encode(&ktypes.UpkeepObservation{
			BlockKey:          obs.BlockKey,
			UpkeepIdentifiers: obs.UpkeepIdentifiers[:i+1],
		})
		if err != nil {
			return nil, err
		}
		if len(b) > limit {
			break
		}
		res = b
	}

	return res, nil
}

func upkeepKeysToString(keys []ktypes.UpkeepKey) string {
	keysStr := make([]string, len(keys))
	for i, key := range keys {
		keysStr[i] = key.String()
	}

	return strings.Join(keysStr, ", ")
}

func createBatches[T any](b []T, size int) (batches [][]T) {
	for i := 0; i < len(b); i += size {
		j := i + size
		if j > len(b) {
			j = len(b)
		}
		batches = append(batches, b[i:j])
	}
	return
}

// buffer is a goroutine safe bytes.Buffer
type buffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *buffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (s *buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}
