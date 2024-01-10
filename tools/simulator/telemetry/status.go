package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
)

func NewUpkeepStatusCollector(path string, verbose bool) *UpkeepStatusCollector {
	err := os.MkdirAll(path, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	return &UpkeepStatusCollector{
		baseCollector: baseCollector{
			t:        UpkeepStatusType,
			io:       []io.WriteCloser{},
			ioLookup: make(map[string]int),
		},
		filePath: path,
		nodes:    make(map[string]*WrappedUpkeepStatusCollector),
		verbose:  verbose,
	}
}

type UpkeepStatusCollector struct {
	baseCollector
	filePath string
	nodes    map[string]*WrappedUpkeepStatusCollector
	verbose  bool
}

func (c *UpkeepStatusCollector) AddNode(node string) error {
	var path string

	if c.verbose {
		path = fmt.Sprintf("%s/%s", c.filePath, node)

		if err := os.MkdirAll(path, 0750); err != nil && !os.IsExist(err) {
			panic(err)
		}
	}

	var file io.WriteCloser

	if !c.verbose {
		file = writeCloseDiscard{}
	} else {
		var perms fs.FileMode = 0666

		flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC

		var err error

		file, err = os.OpenFile(fmt.Sprintf("%s/upkeep_telemetry.log", path), flag, perms)
		if err != nil {
			file.Close()

			return err
		}
	}

	c.ioLookup[node] = len(c.io)
	c.io = append(c.io, file)

	c.nodes[node] = &WrappedUpkeepStatusCollector{
		writer: file,
		data:   make([][]byte, 0, 1000),
	}

	return nil
}

func (c *UpkeepStatusCollector) addWriterForKey(path, key string) error {
	if !c.verbose {
		c.ioLookup[key] = len(c.io)
		c.io = append(c.io, writeCloseDiscard{})

		return nil
	}

	var perms fs.FileMode = 0666

	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	file, err := os.OpenFile(path, flag, perms)
	if err != nil {
		file.Close()

		return err
	}

	c.ioLookup[key] = len(c.io)
	c.io = append(c.io, file)

	return nil
}
func (c *UpkeepStatusCollector) Writer(node string) *WrappedUpkeepStatusCollector {
	collector, exists := c.nodes[node]
	if !exists {
		panic("node does not exist")
	}

	return collector
}

func (c *UpkeepStatusCollector) PrintTabularResults() string {
	tw := table.NewWriter()
	tw.SetTitle("Status Delays per Work ID")
	tw.AppendHeader(table.Row{
		"ID",
		"Surfaced",
		"Proposed",
		"Proposal Quorum",
		"Result Proposed",
		"Result Quorum",
		"Reported",
		"Completed",
	})

	points, err := c.parseData()
	if err != nil {
		panic(err)
	}
	data := c.collapseData(points)

	for _, upkeep := range data {
		tw.AppendRow(
			table.Row{
				shorten(upkeep.ID, 8),
				upkeep.Count(telemetry.Surfaced),
				formatPoint(upkeep, telemetry.Proposed),
				formatPoint(upkeep, telemetry.AgreedInQuorum),
				formatPoint(upkeep, telemetry.ResultProposed),
				formatPoint(upkeep, telemetry.ResultAgreedInQuorum),
				formatPoint(upkeep, telemetry.Reported),
				formatPoint(upkeep, telemetry.Completed),
			})
	}

	return tw.Render()
}

func (c *UpkeepStatusCollector) collapseData(points [][]statusPoint) map[string]*upkeepCompletion {
	mapper := make(map[string]*upkeepCompletion)

	for _, nodePoints := range points {
		for _, point := range nodePoints {
			com, exists := mapper[point.ID]
			if !exists {
				com = newUpkeepCompletion(point.ID)

				com.Add(point)

				mapper[point.ID] = com
			}

			com.Add(point)
		}
	}

	return mapper
}

func (c *UpkeepStatusCollector) parseData() ([][]statusPoint, error) {
	points := [][]statusPoint{}

	for _, node := range c.nodes {
		nodePoints := []statusPoint{}

		for _, data := range node.data {
			var point statusPoint

			if err := json.Unmarshal(data, &point); err != nil {
				return nil, err
			}

			nodePoints = append(nodePoints, point)
		}

		points = append(points, nodePoints)

		// break
	}

	return points, nil
}

func formatPoint(upkeep *upkeepCompletion, status telemetry.Status) string {
	return fmt.Sprintf("%d:%s", upkeep.Count(status), upkeep.Between(telemetry.Surfaced, status))
}

type WrappedUpkeepStatusCollector struct {
	writer io.Writer
	data   [][]byte
}

func (c *WrappedUpkeepStatusCollector) Write(data []byte) (int, error) {
	c.data = append(c.data, data)

	return c.writer.Write(data)
}

type statusPoint struct {
	ID     string   `json:"key"`
	Block  uint64   `json:"block"`
	Status int      `json:"status"`
	Time   NanoTime `json:"time"`
}

type NanoTime time.Time

func (t *NanoTime) UnmarshalJSON(data []byte) error {

	var raw string

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	tm, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return err
	}

	*t = NanoTime(tm)

	return nil
}

func newUpkeepCompletion(id string) *upkeepCompletion {
	return &upkeepCompletion{
		ID:        id,
		statusMap: make(map[int][]time.Time),
	}
}

type upkeepCompletion struct {
	ID        string
	statusMap map[int][]time.Time
}

func (c *upkeepCompletion) Add(point statusPoint) {
	times, exist := c.statusMap[point.Status]
	if !exist {
		times = []time.Time{}
	}

	times = append(times, time.Time(point.Time))

	c.statusMap[point.Status] = times
}

func (c *upkeepCompletion) Count(status telemetry.Status) int {
	times, exist := c.statusMap[int(status)]
	if !exist {
		return 0
	}

	return len(times)
}

func (c *upkeepCompletion) Earliest(status telemetry.Status) (time.Time, error) {
	times, exist := c.statusMap[int(status)]
	if !exist || len(times) == 0 {
		return time.Now(), fmt.Errorf("status does not exist")
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i].Compare(times[j]) < 0
	})

	return times[0], nil
}

func (c *upkeepCompletion) Latest(status telemetry.Status) (time.Time, error) {
	times, exist := c.statusMap[int(status)]
	if !exist || len(times) == 0 {
		return time.Now(), fmt.Errorf("status does not exist")
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i].Compare(times[j]) > 0
	})

	return times[0], nil
}

func (c *upkeepCompletion) Between(start, end telemetry.Status) time.Duration {
	startTime, err := c.Earliest(start)
	if err != nil {
		return 0
	}

	endTime, err := c.Latest(end)
	if err != nil {
		return 0
	}

	return endTime.Sub(startTime)
}

func shorten(full string, outLen int) string {
	if utf8.RuneCountInString(full) < outLen {
		return full
	}

	return string([]byte(full)[:outLen])
}
