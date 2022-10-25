package simulators

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/smartcontractkit/ocr2keepers/cmd/test/config"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSimulatedUpkeeps(t *testing.T) {
	rb := config.RunBook{
		BlockCadence: config.Blocks{
			Genesis:  big.NewInt(1),
			Duration: 100,
		},
		Upkeeps: []config.Upkeep{
			{Count: 15, StartID: big.NewInt(200), GenerateFunc: "12.56x - 3.24", OffsetFunc: "2x - 4"},
		},
	}

	gu, err := GenerateSimulatedUpkeeps(rb)
	assert.NoError(t, err)
	assert.Len(t, gu, 15)
}

func TestGenerateEligibles(t *testing.T) {
	up := SimulatedUpkeep{}
	err := generateEligibles(&up, big.NewInt(9), big.NewInt(50), "4x + 5")
	expected := []int64{14, 18, 22, 26, 30, 34, 38, 42, 46}

	s := []int64{}
	for _, v := range up.EligibleAt {
		s = append(s, v.Int64())
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, s)
}

func TestOperate(t *testing.T) {
	tests := []struct {
		Name string
		A    int64
		B    int64
		Op   string
		ExpZ int64
	}{
		{Name: "Addition", A: 1, B: 4, Op: "+", ExpZ: 5},
		{Name: "Multiplication", A: 3, B: 4, Op: "*", ExpZ: 12},
		{Name: "Subtraction", A: 4, B: 2, Op: "-", ExpZ: 2},
	}

	for _, test := range tests {
		a := decimal.NewFromInt(test.A)
		b := decimal.NewFromInt(test.B)

		z := operate(a, b, test.Op)

		assert.Equal(t, decimal.NewFromInt(test.ExpZ), z)
	}
}
