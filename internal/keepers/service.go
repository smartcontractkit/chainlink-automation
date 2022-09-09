package keepers

import (
	"context"
	"log"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type simpleUpkeepService struct {
	logger   *log.Logger
	ratio    SampleRatio
	registry types.Registry
	shuffler Shuffler[types.UpkeepKey]
	mu       sync.Mutex
	state    map[string]types.UpkeepState
}

// NewSimpleUpkeepService provides an object that implements the UpkeepService in a very
// rudamentary way. Sampling upkeeps is done on demand and completes in linear time with upkeeps.
//
// DO NOT USE THIS IN PRODUCTION
func NewSimpleUpkeepService(ratio SampleRatio, registry types.Registry, logger *log.Logger) *simpleUpkeepService {
	return &simpleUpkeepService{
		logger:   logger,
		ratio:    ratio,
		registry: registry,
		shuffler: new(cryptoShuffler[types.UpkeepKey]),
		state:    make(map[string]types.UpkeepState),
	}
}

var _ UpkeepService = (*simpleUpkeepService)(nil)

func (s *simpleUpkeepService) SampleUpkeeps(ctx context.Context) ([]*types.UpkeepResult, error) {
	// - get all upkeeps from contract
	keys, err := s.registry.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		// TODO: do better error bubbling
		return nil, err
	}

	// - select x upkeeps at random from set
	keys = s.shuffler.Shuffle(keys)
	size := s.ratio.OfInt(len(keys))

	// - check upkeeps selected
	result := []*types.UpkeepResult{}
	for i := 0; i < size; i++ {
		// skip if reported
		s.mu.Lock()
		state, stateSaved := s.state[string(keys[i])]
		if stateSaved && state == Reported {
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()

		// TODO: handle errors correctly
		s.logger.Printf("checking upkeep %s", keys[i])
		ok, u, _ := s.registry.CheckUpkeep(ctx, types.Address([]byte{}), keys[i])
		if ok {
			result = append(result, &u)
		}
	}

	// - return array of results
	return result, nil
}

func (s *simpleUpkeepService) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (types.UpkeepResult, error) {
	// check upkeep at block number in key
	// return result including performData
	// TODO: which address should be passed to this function?
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
