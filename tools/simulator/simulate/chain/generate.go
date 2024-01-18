package chain

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/Maldris/mathparse"
	"github.com/shopspring/decimal"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
)

var (
	ErrUpkeepGeneration = fmt.Errorf("failed to generate upkeep")
)

func GenerateAllUpkeeps(plan config.SimulationPlan) ([]SimulatedUpkeep, error) {
	generated := make([]SimulatedUpkeep, 0)
	limit := new(big.Int).Add(plan.Blocks.Genesis, big.NewInt(int64(plan.Blocks.Duration)))

	for idx, event := range plan.GenerateUpkeeps {
		simulated, err := generateSimulatedUpkeeps(event, plan.Blocks.Genesis, limit)
		if err != nil {
			return nil, fmt.Errorf("%w at index %d", err, idx)
		}

		generated = append(generated, simulated...)
	}

	return generated, nil
}

func GenerateLogTriggers(plan config.SimulationPlan) ([]SimulatedLog, error) {
	logs := make([]SimulatedLog, len(plan.LogEvents))

	for idx, event := range plan.LogEvents {
		logs[idx] = SimulatedLog{
			TriggerAt:    event.TriggerBlock,
			TriggerValue: event.TriggerValue,
		}
	}

	return logs, nil
}

func generateSimulatedUpkeeps(event config.GenerateUpkeepEvent, start *big.Int, limit *big.Int) ([]SimulatedUpkeep, error) {
	applyFunctions := event.EligibilityFunc != "always" && event.EligibilityFunc != "never"

	if !applyFunctions {
		simulated, err := generateBasicSimulatedUpkeeps(event, event.EligibilityFunc == "always")
		if err != nil {
			return nil, err
		}

		return simulated, nil
	}

	simulated, err := generateEligibilityFuncSimulatedUpkeeps(event, start, limit)
	if err != nil {
		return nil, err
	}

	return simulated, nil
}

func generateBasicSimulatedUpkeeps(event config.GenerateUpkeepEvent, alwaysEligible bool) ([]SimulatedUpkeep, error) {
	simulationType, pluginTriggerType, err := getTriggerType(event.UpkeepType)
	if err != nil {
		return nil, err
	}

	generated := make([]SimulatedUpkeep, 0)

	for y := 1; y <= event.Count; y++ {
		id := new(big.Int).Add(event.StartID, big.NewInt(int64(y)))
		simulated := SimulatedUpkeep{
			ID:             id,
			Type:           simulationType,
			CreateInBlock:  event.TriggerBlock,
			UpkeepID:       newUpkeepID(id.Bytes(), pluginTriggerType),
			AlwaysEligible: alwaysEligible,
			EligibleAt:     make([]*big.Int, 0),
			TriggeredBy:    event.LogTriggeredBy,
			Expected:       event.Expected == config.AllExpected,
		}

		generated = append(generated, simulated)
	}

	return generated, nil
}

func generateEligibilityFuncSimulatedUpkeeps(event config.GenerateUpkeepEvent, start *big.Int, limit *big.Int) ([]SimulatedUpkeep, error) {
	simulationType, pluginTriggerType, err := getTriggerType(event.UpkeepType)
	if err != nil {
		return nil, err
	}

	generated := make([]SimulatedUpkeep, 0)
	offset := mathparse.NewParser(event.OffsetFunc)

	offset.Resolve()

	for y := 1; y <= event.Count; y++ {
		id := new(big.Int).Add(event.StartID, big.NewInt(int64(y)))
		sym := SimulatedUpkeep{
			ID:             id,
			Type:           simulationType,
			CreateInBlock:  event.TriggerBlock,
			UpkeepID:       newUpkeepID(id.Bytes(), pluginTriggerType),
			AlwaysEligible: false,
			EligibleAt:     make([]*big.Int, 0),
			TriggeredBy:    event.LogTriggeredBy,
			Expected:       event.Expected == config.AllExpected,
		}

		var genesis *big.Int
		if offset.FoundResult() {
			// create upkeep at id == result
			genesis = big.NewInt(int64(offset.GetValueResult()))
		} else {
			// create upkeep genesis relative to upkeep count
			g, err := calcFromTokens(offset.GetTokens(), big.NewInt(int64(y)))
			if err != nil {
				return nil, err
			}

			genesis = new(big.Int).Add(start, g.BigInt())
		}

		if err := generateEligibles(&sym, genesis, limit, event.EligibilityFunc); err != nil {
			return nil, err
		}

		generated = append(generated, sym)
	}

	return generated, nil
}

func getTriggerType(configType config.UpkeepType) (UpkeepType, uint8, error) {
	switch configType {
	case config.ConditionalUpkeepType:
		return ConditionalType, uint8(types.ConditionTrigger), nil
	case config.LogTriggerUpkeepType:
		return LogTriggerType, uint8(types.LogTrigger), nil
	default:
		return 0, 0, fmt.Errorf("%w: trigger type '%s' unrecognized", ErrUpkeepGeneration, configType)
	}
}

func operate(a, b decimal.Decimal, op string) decimal.Decimal {
	switch op {
	case "+":
		return a.Add(b)
	case "*":
		return a.Mul(b)
	case "-":
		return a.Sub(b)
	default:
	}

	return decimal.Zero
}

func generateEligibles(upkeep *SimulatedUpkeep, genesis *big.Int, limit *big.Int, f string) error {
	p := mathparse.NewParser(f)
	p.Resolve()

	if p.FoundResult() {
		return fmt.Errorf("simple value unsupported")
	} else {
		// create upkeep from offset function
		var i int64 = 0
		nextValue := big.NewInt(0)
		tokens := p.GetTokens()

		for nextValue.Cmp(limit) < 0 {
			if nextValue.Cmp(genesis) >= 0 {
				upkeep.EligibleAt = append(upkeep.EligibleAt, nextValue)
			}

			value, err := calcFromTokens(tokens, big.NewInt(i))
			if err != nil {
				return err
			}

			biValue := value.Round(0).BigInt()
			nextValue = new(big.Int).Add(genesis, biValue)
			i++
		}
	}

	return nil
}

func calcFromTokens(tokens []mathparse.Token, x *big.Int) (decimal.Decimal, error) {
	value := decimal.NewFromInt(0)
	action := "+"

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token.Type {
		case 2, 3:
			var tVal decimal.Decimal

			if token.Value == "x" {
				tVal = decimal.NewFromBigInt(x, int32(0))
			} else {
				tVal = decimal.NewFromFloat(token.ParseValue)
			}

			value = operate(value, tVal, action)
		case 4:
			action = token.Value
		// case 1, 5, 6, 7, 8:
		// log.Printf("unused token: %s", token.Value)
		default:
		}
	}

	return value, nil
}

func newUpkeepID(entropy []byte, uType uint8) [32]byte {
	/*
	   Following the contract convention, an identifier is composed of 32 bytes:

	   - 4 bytes of entropy
	   - 11 bytes of zeros
	   - 1 identifying byte for the trigger type
	   - 16 bytes of entropy
	*/
	hashedValue := sha256.Sum256(entropy)

	for x := 4; x < 15; x++ {
		hashedValue[x] = uint8(0)
	}

	hashedValue[15] = uType

	return hashedValue
}
