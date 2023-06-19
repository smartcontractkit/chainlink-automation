package instructions

type Instruction string

const (
	ShouldCoordinateBlock Instruction = "should coordinate block"
	DoCoordinateBlock     Instruction = "do coordinate block"
)
