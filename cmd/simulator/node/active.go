package node

type Closable interface {
	Close() error
}

type Simulator struct {
	Service Closable
}
