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
	verbose  bool
}

func NewNodeLogCollector(path string, verbose bool) *NodeLogCollector {
	if verbose {
		if err := os.MkdirAll(path, 0750); err != nil && !os.IsExist(err) {
			panic(err)
		}
	}

	return &NodeLogCollector{
		baseCollector: baseCollector{
			t:        NodeLogType,
			io:       []io.WriteCloser{},
			ioLookup: make(map[string]int),
		},
		filePath: path,
		verbose:  verbose,
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

	if c.verbose {
		if err := os.MkdirAll(path, 0750); err != nil && !os.IsExist(err) {
			panic(err)
		}
	}

	if err := c.addWriterForKey(fmt.Sprintf("%s/general.log", path), fmt.Sprintf("general/%s", node)); err != nil {
		return err
	}

	if err := c.addWriterForKey(fmt.Sprintf("%s/contract.log", path), fmt.Sprintf("contract/%s", node)); err != nil {
		return err
	}

	return nil
}

func (c *NodeLogCollector) addWriterForKey(path, key string) error {
	if !c.verbose {
		c.ioLookup[key] = len(c.io)
		c.io = append(c.io, writeCloseDiscard{})

		return nil
	}

	var perms fs.FileMode = 0666

	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	file, err := os.OpenFile(fmt.Sprintf("%s/general.log", path), flag, perms)
	if err != nil {
		file.Close()

		return err
	}

	c.ioLookup[key] = len(c.io)
	c.io = append(c.io, file)

	return nil
}

type writeCloseDiscard struct{}

func (writeCloseDiscard) Write(bts []byte) (int, error) {
	return len(bts), nil
}

func (writeCloseDiscard) Close() error {
	return nil
}
