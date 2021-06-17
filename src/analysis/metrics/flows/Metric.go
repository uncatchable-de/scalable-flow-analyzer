package flows

import (
	"analysis/flows"
	"analysis/metrics/common"
	"encoding/json"
	"fmt"
	"github.com/dustin/go-humanize"
	"os"
	"path"
	"time"
)

type Metric struct {
	computeRRPs  bool
	rrIdentifier *common.ReqResIdentifier

	exportChannel chan *string
	doneChannel   chan bool

	metrics   []registrableMetric
	rrMetrics []registrableRRMetric
}

type ExportableValue interface {
	export() map[string]interface{}
}

type registrableMetric interface {
	onFlush(flow *flows.Flow) ExportableValue
}

type registrableRRMetric interface {
	onFlush(flow *flows.Flow, reqRes []*common.RequestResponse) ExportableValue
}

func NewMetric(samplingRate int64, computeRRPs bool, exportBufferSize uint) *Metric {
	metric := &Metric{
		computeRRPs:   computeRRPs,
		exportChannel: make(chan *string, exportBufferSize),
		doneChannel:   make(chan bool),
	}

	metricFlowRate := newMetricFlowRate()
	metricFlowRate.samplingRate = samplingRate

	metric.addMetric(metricFlowRate)
	metric.addMetric(newMetricProtocol())
	metric.addMetric(newMetricFlowSize())
	metric.addMetric(newMetricPackets())
	metric.addMetric(newMetricFlowDuration())

	if !computeRRPs {
		return metric
	}

	metric.rrIdentifier = common.NewReqResIdentifier(
		false, false,
		nil, nil,
	)

	metric.addRRMetric(newMetricRRPs())

	return metric
}

func (m *Metric) addMetric(metric registrableMetric) {
	m.metrics = append(m.metrics, metric)
}

func (m *Metric) addRRMetric(rrMetric registrableRRMetric) {
	m.rrMetrics = append(m.rrMetrics, rrMetric)
}

// Callback that is called by the pools, once reconstruction for a flow is done.
// This means that this method runs concurrently.
func (m *Metric) OnTCPFlush(flow *flows.TCPFlow) {
	var protocol = common.GetProtocol(&flow.Flow)
	var rr = make([]*common.RequestResponse, 0)
	var dropFlow bool

	if m.computeRRPs {
		rr, dropFlow = m.rrIdentifier.OnTCPFlush(protocol, flow)
		if dropFlow {
			return
		}
	}

	m.onFlush(&flow.Flow, rr)
}

// Callback that is called by the pools, once reconstruction for a flow is done.
// This means that this method runs concurrently.
func (m *Metric) OnUDPFlush(flow *flows.UDPFlow) {
	var protocol = common.GetProtocol(&flow.Flow)
	var rr = make([]*common.RequestResponse, 0)
	var dropFlow bool

	if m.computeRRPs {
		rr, dropFlow = m.rrIdentifier.OnUDPFlush(protocol, flow)
		if dropFlow {
			return
		}
	}

	m.onFlush(&flow.Flow, rr)
}

// This method is called by the callback. Simplifies metric implementation, as
// they are not required to implement different methods for TCP/UDP.
func (m *Metric) onFlush(flow *flows.Flow, rr []*common.RequestResponse) {
	values := make([]ExportableValue, len(m.metrics)+len(m.rrMetrics))

	for i, metric := range m.metrics {
		values[i] = metric.onFlush(flow)
	}

	if m.computeRRPs {
		for i, rrMetric := range m.rrMetrics {
			values[len(m.metrics)+i] = rrMetric.onFlush(flow, rr)
		}
	}

	combinedMetric := combineMetrics(values)
	m.exportChannel <- serializeMetric(combinedMetric)
}

// Combines metrics that have been computed independently into one.
func combineMetrics(values []ExportableValue) *map[string]interface{} {
	combinedMetric := make(map[string]interface{})

	for _, value := range values {
		for key, value := range value.export() {
			combinedMetric[key] = value
		}
	}

	return &combinedMetric
}

// Serializes a combined metric into a JSON string.
func serializeMetric(metric *map[string]interface{}) *string {
	b, err := json.Marshal(metric)
	if err != nil {
		fmt.Println(err.Error())
		panic("Error during json marshalling!")
	}

	serialized := string(b)
	return &serialized
}

// Closes the exportChannel, which causes all buffered metrics to be flushed.
func (m *Metric) Flush() {
	close(m.exportChannel)
}

// Waits until all metrics have been written to file.
func (m *Metric) Wait() {
	<-m.doneChannel
}

// Should always be called as a goroutine. Writes serialized metrics directly to disk.
func (m *Metric) ExportRoutine(directory string) {
	filename := path.Join(directory, "flow_metrics.json")
	if _, err := os.Stat(filename); err == nil {
		// File exists
		err := os.Remove(filename)
		if err != nil {
			fmt.Println(err.Error())
			panic("Could not remove '" + filename + "'!")
		}
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Println(err.Error())
		panic("Could not create '" + filename + "'!")
	}

	err = os.Chmod(filename, 0644)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Could not change permissions for '" + filename + "'!")
	}

	fmt.Println("Export routine successfully setup.")
	start := time.Now()

	_, err = f.WriteString("{")
	if err != nil {
		fmt.Println(err.Error())
		panic("Error writing to file!")
	}

	id := 0
	serializedMetric := ""
	for serializedMetricPointer := range m.exportChannel {
		if len(serializedMetric) != 0 {
			_, err = f.WriteString(fmt.Sprintf("\"%d\":%s,", id, serializedMetric))
			if err != nil {
				fmt.Println(err.Error())
				panic("Error writing to file!")
			}
		}

		serializedMetric = *serializedMetricPointer
		id++
	}

	_, err = f.WriteString(fmt.Sprintf("\"%d\":%s}", id, serializedMetric))
	if err != nil {
		fmt.Println(err.Error())
		panic("Error writing to file!")
	}

	err = f.Close()
	if err != nil {
		fmt.Println(err.Error())
		panic("Error closing file!")
	}

	fmt.Println("Finished writing json. Took:\t", time.Since(start))
	fmt.Printf("Export successful. Exported:\t %s flow metrics", humanize.Comma(int64(id)))

	m.doneChannel <- true
	close(m.doneChannel)
}
