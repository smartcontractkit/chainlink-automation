package telemetry

import (
	"io"
)

type Collector interface {
	Type() CollectorType
	Close() error
}

type CollectorType int

const (
	RPCType CollectorType = iota
	NodeLogType
	UpkeepStatusType
)

type baseCollector struct {
	t        CollectorType
	io       []io.WriteCloser
	ioLookup map[string]int
}

func (c *baseCollector) Type() CollectorType {
	return c.t
}

func (c *baseCollector) Close() error {
	for _, w := range c.io {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}
