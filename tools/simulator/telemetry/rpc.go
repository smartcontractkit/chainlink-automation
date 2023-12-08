package telemetry

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type RPCCollector struct {
	baseCollector
	filePath string
	verbose  bool
	nodes    map[string]*WrappedRPCCollector
}

func NewNodeRPCCollector(path string, verbose bool) *RPCCollector {
	return &RPCCollector{
		baseCollector: baseCollector{
			t: RPCType,
		},
		verbose:  verbose,
		filePath: path,
		nodes:    make(map[string]*WrappedRPCCollector),
	}
}

func (c *RPCCollector) AddNode(node string) error {
	wc := &WrappedRPCCollector{
		rate:  []int{},
		calls: []rpcDataPoint{},
	}

	c.nodes[node] = wc

	return nil
}

func (c *RPCCollector) WriteResults() error {
	if !c.verbose {
		return nil
	}

	for key, node := range c.nodes {
		path := fmt.Sprintf("%s/%s", c.filePath, key)
		err := os.MkdirAll(path, 0750)
		if err != nil && !os.IsExist(err) {
			panic(err)
		}

		b, err := json.Marshal(node.rate)
		if err != nil {
			return err
		}

		// write JSON file for rate data
		err = c.writeDataToFile(fmt.Sprintf("%s/rpc_call_rate.json", path), b)
		if err != nil {
			return err
		}

		b, err = json.Marshal(node.calls)
		if err != nil {
			return err
		}

		// write JSON file for call detail data
		err = c.writeDataToFile(fmt.Sprintf("%s/rpc_call_detail.json", path), b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *RPCCollector) writeDataToFile(path string, data []byte) error {
	var perms fs.FileMode = 0666
	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC

	f, err := os.OpenFile(path, flag, perms)
	if err != nil {
		f.Close()
		return err
	}

	defer f.Close()

	_, err = f.Write(data)
	return err
}

func (c *RPCCollector) RPCCollectorNode(node string) *WrappedRPCCollector {
	wc, ok := c.nodes[node]
	if !ok {
		panic("node not available")
	}

	return wc
}

func (c *RPCCollector) SummaryChart() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		page := components.NewPage()
		page.SetLayout(components.PageFlexLayout)
		nodeIDs, points, err := c.collectDataFromFile()
		if err != nil {
			panic(err)
		}

		for _, nodeID := range nodeIDs {
			data := buildChartData(points[nodeID])
			line := charts.NewLine()
			// set some global options like Title/Legend/ToolTip or anything else
			line.SetGlobalOptions(
				charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
				charts.WithTitleOpts(opts.Title{
					Title:    "RPC Requests per Second",
					Subtitle: fmt.Sprintf("node: %s", nodeID),
				}),
				charts.WithXAxisOpts(opts.XAxis{Name: "time (10s)", Type: "category"}),
				charts.WithYAxisOpts(opts.YAxis{Type: "value", AxisLabel: &opts.AxisLabel{Rotate: 90}}),
				charts.WithToolboxOpts(opts.Toolbox{Show: true}),
				charts.WithLegendOpts(opts.Legend{Left: "center", Top: "top"}))

			xLabels := data.CallRateLabels

			// Put data into instance
			line.SetXAxis(xLabels).
				AddSeries(nodeID, generateLineItems(data.CallRate)).
				AddSeries(nodeID, generateLineItems(data.ErrorRate), charts.WithLineStyleOpts(opts.LineStyle{Color: "red"})).
				AddSeries(nodeID, generateLineItems(data.ContextCancels), charts.WithLineStyleOpts(opts.LineStyle{Color: "yellow"})).
				AddSeries(nodeID, generateLineItems(data.RateLimits), charts.WithLineStyleOpts(opts.LineStyle{Color: "blue"})).
				SetSeriesOptions(charts.WithLineChartOpts(opts.LineChart{Smooth: true}))

			box := charts.NewBoxPlot()
			box.SetGlobalOptions(
				charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
				charts.WithTitleOpts(opts.Title{
					Title:    "RPC Latency",
					Subtitle: fmt.Sprintf("node: %s", nodeID),
				}),
			)
			box.SetXAxis(xLabels).AddSeries("boxplot", generateBoxPlotItems(data.Latency))

			page.AddCharts(line, box)
		}

		_ = page.Render(w)
	}
}

func (c *RPCCollector) collectDataFromFile() ([]string, map[string][]rpcDataPoint, error) {
	ids := []string{}
	rateData := make(map[string][]rpcDataPoint)
	path := c.filePath

	if !c.verbose {
		return ids, rateData, fmt.Errorf("verbose logging not activated")
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return ids, rateData, err
	}

	for _, file := range files {
		if file.IsDir() {
			nm := file.Name()

			dataPath := fmt.Sprintf("%s/%s/rpc_call_detail.json", path, nm)
			b, err := os.ReadFile(dataPath)
			if err != nil {
				return ids, rateData, err
			}

			var data []rpcDataPoint
			err = json.Unmarshal(b, &data)
			if err != nil {
				return ids, rateData, err
			}

			ids = append(ids, nm)
			rateData[nm] = data
		}
	}

	sort.Strings(ids)

	return ids, rateData, nil
}

func buildChartData(points []rpcDataPoint) nodeStats {
	stats := nodeStats{
		CallRateLabels: []string{},
		CallRate:       []int{},
		ErrorRate:      []int{},
		ContextCancels: []int{},
		RateLimits:     []int{},
	}

	if len(points) == 0 {
		return stats
	}

	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Timestamp.Before(points[j].Timestamp)
	})

	idx := -1
	timeReference := time.Now().Add(-1_000_000 * time.Hour)
	for _, point := range points {
		if point.Timestamp.Sub(timeReference) >= 10*time.Second {
			timeReference = point.Timestamp
			// set new data points
			stats.CallRateLabels = append(stats.CallRateLabels, fmt.Sprintf("%d", idx+1))
			stats.CallRate = append(stats.CallRate, 0)
			stats.ErrorRate = append(stats.ErrorRate, 0)
			stats.ContextCancels = append(stats.ContextCancels, 0)
			stats.RateLimits = append(stats.RateLimits, 0)
			stats.Latency = append(stats.Latency, []int{})
			// increment idx
			idx++
		}

		// increment existing data point
		stats.CallRate[idx]++
		if point.Err != "" {
			stats.ErrorRate[idx]++

			if strings.Contains(point.Err, "context") {
				stats.ContextCancels[idx]++
			}

			if strings.Contains(point.Err, "rate") {
				stats.RateLimits[idx]++
			}
		}

		stats.Latency[idx] = append(stats.Latency[idx], int(point.Latency/time.Millisecond))
	}

	return stats
}

type nodeStats struct {
	CallRateLabels []string
	CallRate       []int
	ErrorRate      []int
	ContextCancels []int
	RateLimits     []int
	Latency        [][]int
}

func generateLineItems(values []int) []opts.LineData {
	items := make([]opts.LineData, len(values))
	for i, value := range values {
		items[i] = opts.LineData{Value: value / 10, XAxisIndex: i}
	}

	return items
}

func generateBoxPlotItems(boxPlotData [][]int) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, 0)
	for i := 0; i < len(boxPlotData); i++ {
		items = append(items, opts.BoxPlotData{Value: boxPlotData[i]})
	}
	return items
}

type WrappedRPCCollector struct {
	mu    sync.Mutex
	rate  []int
	calls []rpcDataPoint
}

func (wc *WrappedRPCCollector) Register(name string, latency time.Duration, err error) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	wc.calls = append(wc.calls, rpcDataPoint{
		Timestamp: time.Now(),
		Name:      name,
		Latency:   latency,
		Err:       errStr,
	})
}

func (wc *WrappedRPCCollector) AddRateDataPoint(calls int) {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	wc.rate = append(wc.rate, calls)
}

type rpcDataPoint struct {
	Timestamp time.Time     `json:"timestamp"`
	Name      string        `json:"rpcCallName"`
	Latency   time.Duration `json:"callLatency"`
	Err       string        `json:"errorMessage"`
}
