package instructions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		instruction Instruction
		err         error
	}{
		{
			name:        "should coordinate",
			instruction: ShouldCoordinateBlock,
		},
		{
			name:        "do coordinate",
			instruction: DoCoordinateBlock,
		},
		{
			name:        "error on anything else",
			instruction: "",
			err:         ErrInvalidInstruction,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.instruction)

			if test.err == nil {
				assert.NoError(t, err, "no error expected")
			} else {
				assert.ErrorIs(t, err, test.err, "is expected error type")
			}
		})
	}
}
