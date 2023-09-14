package telemetry

import (
	"fmt"
	"io"
	"io/fs"
	"os"
)

type NodeLogCollector struct {
	baseCollector
	filePath string
}

func NewNodeLogCollector(path string) *NodeLogCollector {
	err := os.MkdirAll(path, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	return &NodeLogCollector{
		baseCollector: baseCollector{
			t:        NodeLogType,
			io:       []io.WriteCloser{},
			ioLookup: make(map[string]int),
		},
		filePath: path,
	}
}

func (c *NodeLogCollector) ContractLog(node string) io.Writer {
	key := fmt.Sprintf("contract/%s", node)

	idx, ok := c.baseCollector.ioLookup[key]
	if !ok {
		panic(fmt.Errorf("missing contract log for %s", node))
	}

	return c.baseCollector.io[idx]
}

func (c *NodeLogCollector) GeneralLog(node string) io.Writer {
	key := fmt.Sprintf("general/%s", node)

	idx, ok := c.baseCollector.ioLookup[key]
	if !ok {
		panic(fmt.Errorf("missing general log for %s", node))
	}

	return c.baseCollector.io[idx]
}

func (c *NodeLogCollector) AddNode(node string) error {
	path := fmt.Sprintf("%s/%s", c.filePath, node)
	err := os.MkdirAll(path, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	var perms fs.FileMode = 0666
	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	key := fmt.Sprintf("general/%s", node)

	f, err := os.OpenFile(fmt.Sprintf("%s/general.log", path), flag, perms)
	if err != nil {
		f.Close()
		return err
	}

	c.ioLookup[key] = len(c.io)
	c.io = append(c.io, f)

	key = fmt.Sprintf("contract/%s", node)

	cLog, err := os.OpenFile(fmt.Sprintf("%s/contract.log", path), flag, perms)
	if err != nil {
		cLog.Close()
		return err
	}

	c.ioLookup[key] = len(c.io)
	c.io = append(c.io, cLog)

	return nil
}
