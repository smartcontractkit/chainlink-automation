package node

type Closable interface {
	Close() error
}

type Stoppable interface {
	Stop()
}

type Simulator struct {
	Service  Closable
	Contract Stoppable
}
