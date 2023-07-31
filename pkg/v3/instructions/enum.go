package instructions

import "fmt"

type Instruction string

const (
	ShouldCoordinateBlock Instruction = "should coordinate block"
	DoCoordinateBlock     Instruction = "do coordinate block"
)

var (
	ErrInvalidInstruction = fmt.Errorf("invalid instruction")
)

func Validate(i Instruction) error {
	switch i {
	case ShouldCoordinateBlock, DoCoordinateBlock:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidInstruction, i)
	}
}
