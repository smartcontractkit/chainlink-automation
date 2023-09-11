package upkeep

import (
	"math/big"
	"testing"
)

func TestIsConditionalEligible(t *testing.T) {
	tests := []struct {
		name      string
		eligibles []*big.Int
		performs  []*big.Int
		block     *big.Int
		expected  bool
	}{
		{
			name:      "eligible no performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{},
			block:     big.NewInt(15), // block is between first and second eligible
			expected:  true,
		},
		{
			name:      "not eligible no performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20)},
			performs:  []*big.Int{},
			block:     big.NewInt(9), // block is before first eligible
			expected:  false,
		},
		{
			name:      "eligible with 1 perform",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15)},
			block:     big.NewInt(21), // block is between second and third eligible
			expected:  true,
		},
		{
			name:      "not eligible with 1 perform",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15)},
			block:     big.NewInt(18), // block is between first and second eligible
			expected:  false,
		},
		{
			name:      "eligible with 2 performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15), big.NewInt(22)},
			block:     big.NewInt(31), // block is after third eligible
			expected:  true,
		},
		{
			name:      "not eligible with 2 performs",
			eligibles: []*big.Int{big.NewInt(10), big.NewInt(20), big.NewInt(30)},
			performs:  []*big.Int{big.NewInt(15), big.NewInt(22)},
			block:     big.NewInt(26), // block is between second and third eligible
			expected:  false,
		},
	}

	for i := range tests {
		eligible := isConditionalEligible(tests[i].eligibles, tests[i].performs, tests[i].block)

		if eligible != tests[i].expected {
			t.Logf("%s was %t but was not expected", tests[i].name, eligible)
			t.Fail()
		}
	}
}
