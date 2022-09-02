package keepers

import (
	"context"
	"math/rand"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type simpleUpkeepService struct {
	ratio    SampleRatio
	registry types.Registry
	mu       sync.Mutex
	state    map[string]types.UpkeepState
}

// NewSimpleUpkeepService provides an object that implements the UpkeepService in a very
// rudamentary way. Sampling upkeeps is done on demand and completes in linear time with upkeeps.
//
// DO NOT USE THIS IN PRODUCTION
func NewSimpleUpkeepService(ratio SampleRatio, registry types.Registry) *simpleUpkeepService {
	return &simpleUpkeepService{
		ratio:    ratio,
		registry: registry,
		state:    make(map[string]types.UpkeepState),
	}
}

var _ UpkeepService = (*simpleUpkeepService)(nil)

func (s *simpleUpkeepService) SampleUpkeeps(ctx context.Context) ([]types.UpkeepResult, error) {
	// - get all upkeeps from contract
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		// TODO: do better error bubbling
		return nil, err
	}

	// - select x upkeeps at random from set
	rnd := rand.New(newCryptoRandSource())
	rnd.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	size := s.ratio.OfInt(len(keys))

	// - check upkeeps selected
	result := []types.UpkeepResult{}
	for i := 0; i < size; i++ {
		// skip if reported
		s.mu.Lock()
		state, reported := s.state[string(keys[i])]
		if reported && state == Reported {
			continue
		}
		s.mu.Unlock()

		// TODO: handle errors correctly
		ok, u, _ := s.registry.CheckUpkeep(ctx, types.Address([]byte{}), keys[i])
		if ok {
			result = append(result, types.UpkeepResult{Key: keys[i], State: Perform, PerformData: u.PerformData})
		}
	}

	// - return array of results
	return result, nil
}

func (s *simpleUpkeepService) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (types.UpkeepResult, error) {
	// check upkeep at block number in key
	// return result including performData
	ok, u, err := s.registry.CheckUpkeep(ctx, types.Address([]byte{}), key)
	if err != nil {
		// TODO: do better error bubbling
		return types.UpkeepResult{}, err
	}

	result := types.UpkeepResult{
		Key:   key,
		State: Skip,
	}

	if ok {
		result.State = Perform
		result.PerformData = u.PerformData
	}

	return result, nil
}

func (s *simpleUpkeepService) SetUpkeepState(_ context.Context, uk types.UpkeepKey, state types.UpkeepState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[string(uk)] = state
	return nil
}
