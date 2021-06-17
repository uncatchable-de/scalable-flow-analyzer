package standard

import (
	"analysis/flows"
	"analysis/metrics/common"
	"fmt"
	"time"
)

type MetricFlowRate struct {
	flowRates common.IntMetricUnivariate
}

func newMetricFlowRate() *MetricFlowRate {
	var metricFlowRates = MetricFlowRate{}
	metricFlowRates.flowRates = common.NewIntMetricUnivariate(1, false)
	return &metricFlowRates
}

func (mfr *MetricFlowRate) calc(flow *flows.Flow) (flowRates []int) {
	var size int
	var start, end int64

	packets := flow.Packets
	for i := 0; i < len(packets); i++ {
		p := packets[i]

		size += int(p.LengthPayload)
		if start == 0 {
			start = p.Timestamp
		}
		end = p.Timestamp
	}

	seconds := int((end - start) / time.Second.Nanoseconds())
	if seconds == 0 {
		seconds = 1
	}

	flowRates = append(flowRates, size/seconds)
	return flowRates
}

func (mfr *MetricFlowRate) OnTCPFlush(flow *flows.TCPFlow) {
	mfr.onFlush(&(flow.Flow))
}

func (mfr *MetricFlowRate) OnUDPFlush(flow *flows.UDPFlow) {
	mfr.onFlush(&(flow.Flow))
}

func (mfr *MetricFlowRate) onFlush(flow *flows.Flow) {
	flowRates := mfr.calc(flow)
	if len(flowRates) > 0 {
		protocol := common.GetProtocol(flow)
		mfr.flowRates.AddValue(protocol, DefaultClusterIndex, flowRates...)
	}
}

// Export returns the metric data per Protocol
func (mfr *MetricFlowRate) Export(protocolKey common.ProtocolKeyType) *common.ExportUnivariateFormat {
	return mfr.flowRates.Export(protocolKey, DefaultClusterIndex)
}

// Export the stored protocols
func (mfr *MetricFlowRate) GetProtocols() []common.Protocol {
	return mfr.flowRates.GetProtocols()
}

func (mfr *MetricFlowRate) Name() string {
	return "FlowRate"
}

func (mfr *MetricFlowRate) PrintStatistic(verbose bool) {
	fmt.Println("Metric Flow rates:")
	fmt.Print(mfr.flowRates.GetStatistics(verbose))
}
