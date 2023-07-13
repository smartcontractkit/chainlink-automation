package instruction

type Instruction string

type InstructionStore interface {
	Set(key Instruction)
	Has(key Instruction) bool
	Delete(key Instruction)
}
