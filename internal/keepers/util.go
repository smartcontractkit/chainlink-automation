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

func shuffledDedupedKeyList(
	attributed []types.AttributedObservation,
	key [16]byte,
	totalLimit int,
	filters ...func(ktypes.UpkeepKey) bool,
) ([]ktypes.UpkeepKey, error) {
	var err error

	if len(attributed) == 0 {
		return nil, fmt.Errorf("%w: must provide at least 1", ErrNotEnoughInputs)
	}

	var parseErrors int
	kys := make([][]ktypes.UpkeepKey, len(attributed))

	var allBlockKeys []ktypes.BlockKey
	for i, attr := range attributed {
		b := []byte(attr.Observation)
		if len(b) == 0 {
			continue
		}

		// a single observation returning an error here can void all other
		// good observations. ensure this loop continues on error, but collect
		// them and throw an error if ALL observations fail at this point.
		// TODO we can't rely on this concrete type for decoding/encoding
		var upkeepObservation *ktypes.UpkeepObservation

		err = decode(b, &upkeepObservation)
		if err != nil {
			parseErrors++
			continue
		}

		allBlockKeys = append(allBlockKeys, upkeepObservation.BlockKey)

		if len(upkeepObservation.UpkeepIdentifiers) > 0 {
			kys[i] = []ktypes.UpkeepKey{
				chain.NewUpkeepKeyFromBlockAndID(upkeepObservation.BlockKey, upkeepObservation.UpkeepIdentifiers[0]),
			}
		}
	}

	medianBlock := calculateMedianBlock(allBlockKeys)
	for i, ob := range kys {
		var ks []ktypes.UpkeepKey
		for _, k := range ob {
			_, upkeepID, err := k.BlockKeyAndUpkeepID()
			if err != nil {
				return nil, err
			}
			ks = append(ks, chain.NewUpkeepKeyFromBlockAndID(medianBlock, upkeepID))
		}
		kys[i] = ks
	}

	if parseErrors == len(attributed) {
		return nil, fmt.Errorf("%w: cannot prepare sorted key list; observations not properly encoded", err)
	}

	// Limit number of total keys to "totalLimit" max
	if len(kys) > totalLimit {
		kys = kys[:totalLimit]
	}

	keys, err := dedupe(kys, filters...)
	if err != nil {
		return nil, fmt.Errorf("%w: observation dedupe", err)
	}

	// TODO: a hacky solution assuming upkeep key structure
	// removes duplicate upkeep ids in preference of ids at higher blocks
	// needs to be refactored
	// AUTO-1480
	idxMap := make(map[string]int)
	out := make([]ktypes.UpkeepKey, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		blockKey, upkeepID, err := keys[i].BlockKeyAndUpkeepID()
		if err != nil {
			return nil, err
		}

		idx, ok := idxMap[string(upkeepID)]
		if !ok {
			idxMap[string(upkeepID)] = len(out)
			out = append(out, keys[i])
			continue
		}

		savedBlockKey, _, err := out[idx].BlockKeyAndUpkeepID()
		if err != nil {
			return nil, err
		}

		if string(blockKey) > string(savedBlockKey) {
			out[idx] = keys[i]
		}
	}
	keys = out

	src := util.NewKeyedCryptoRandSource(key)
	r := rnd.New(src)
	r.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	return keys, nil
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
	var b []byte
	var err error

	emptyObservation := &ktypes.UpkeepObservation{
		BlockKey:          obs.BlockKey,
		UpkeepIdentifiers: []ktypes.UpkeepIdentifier{},
	}

	b, err = encode(emptyObservation)
	if err != nil {
		return nil, err
	}

	// calculate how many bytes we have free for the upkeep identifiers by first calculating the size of an upkeep
	// observation containing a block key and an empty list of upkeep identifiers
	// an empty UpkeepObservation with only a block key will look like:
	//   {"blockKey":123,"upkeepIdentifiers":[]}
	// a populated UpkeepObservation with a block key and upkeep identifiers will look like:
	//   {"blockKey":123,"upkeepIdentifiers":[234,567,890]}

	// subtract the size of the "empty" upkeep observation to calculate the limit for the upkeep identifiers
	limit -= len(b)

	// add 2 to the limit since the "[]" brackets for the list of identifiers are included in both the empty and
	// populated observation
	limit += 2

	// limit the number of keys that can be added to an observation
	// OCR observation limit is set to 1_000 bytes so this should be under the
	// limit
	var tot int
	var idx int

	// json encoding byte arrays follows a linear progression of byte length
	// vs encoded length. the following is the magic equation where x is the
	// byte array length and y is the encoded length.
	// eq: y = 1.32 * x + 7.31

	keys := obs.UpkeepIdentifiers
	// if the total plus padding for json encoding is less than the max, another
	// key can be included
	c := true
	for c && idx < len(keys) {
		tot += len(string(keys[idx]))

		// because we are encoding an array of byte arrays, some bytes are added
		// per byte array and all byte arrays could be different lengths.
		// this is only for the purpose of estimation so add some padding for
		// each byte array
		v := (1.32 * float64(tot)) + 7.31 + float64((idx+1)*2)
		if int(math.Ceil(v)) > limit {
			c = false
		}

		idx++
	}

	limitedKeys := keys[:idx]
	emptyObservation.UpkeepIdentifiers = limitedKeys

	b, err = encode(emptyObservation)
	if err != nil {
		return nil, err
	}

	// finally we walk backward from the estimate if the resulting length is
	// larger than our limit. this ensures that the output is within range.
	for len(b) > limit {
		idx--
		if idx == 0 {
			return encode(&ktypes.UpkeepObservation{})
		}

		limitedKeys = keys[:idx]
		emptyObservation.UpkeepIdentifiers = limitedKeys

		b, err = encode(emptyObservation)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
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
