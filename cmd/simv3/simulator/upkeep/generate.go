package upkeep

import (
	"fmt"
	"math/big"

	"github.com/Maldris/mathparse"
	"github.com/shopspring/decimal"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
)

func GenerateConditionals(rb config.RunBook) ([]chain.SimulatedUpkeep, error) {
	generated := []chain.SimulatedUpkeep{}
	limit := new(big.Int).Add(rb.BlockCadence.Genesis, big.NewInt(int64(rb.BlockCadence.Duration)))

	for _, upkeep := range rb.Upkeeps {
		p := mathparse.NewParser(upkeep.OffsetFunc)
		p.Resolve()

		for y := 1; y <= upkeep.Count; y++ {
			sym := chain.SimulatedUpkeep{
				ID:         new(big.Int).Add(upkeep.StartID, big.NewInt(int64(y))),
				EligibleAt: make([]*big.Int, 0),
			}

			var genesis *big.Int
			if p.FoundResult() {
				// create upkeep at id == result
				genesis = big.NewInt(int64(p.GetValueResult()))
			} else {
				// create upkeep genesis relative to upkeep count
				g, err := calcFromTokens(p.GetTokens(), big.NewInt(int64(y)))
				if err != nil {
					return nil, err
				}

				genesis = new(big.Int).Add(rb.BlockCadence.Genesis, g.BigInt())
			}

			if err := generateEligibles(&sym, genesis, limit, upkeep.GenerateFunc); err != nil {
				return nil, err
			}

			generated = append(generated, sym)
		}
	}

	return generated, nil
}

func GenerateLogTriggeredUpkeeps(rb config.RunBook) ([]chain.SimulatedUpkeep, error) {

	return nil, nil
}

func GenerateLogTriggers(rb config.RunBook) ([]chain.SimulatedLog, error) {

	return nil, nil
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

func generateEligibles(upkeep *chain.SimulatedUpkeep, genesis *big.Int, limit *big.Int, f string) error {
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
			if nextValue.Int64() > 0 {
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
