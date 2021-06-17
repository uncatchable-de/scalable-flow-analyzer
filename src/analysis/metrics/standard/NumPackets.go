package standard

import (
	"analysis/flows"
	"analysis/metrics/common"
	"fmt"
)

type MetricNumPackets struct {
	numPackets common.IntMetric
}

func newMetricNumPackets() *MetricNumPackets {
	var metricNumPackets = MetricNumPackets{}
	metricNumPackets.numPackets = common.NewIntMetric()
	return &metricNumPackets
}

func (mnp *MetricNumPackets) OnTCPFlush(flow *flows.TCPFlow) {
	protocol := common.GetProtocol(&(flow.Flow))
	mnp.numPackets.AddValue(protocol, len(flow.Packets))
}

func (mnp *MetricNumPackets) OnUDPFlush(flow *flows.UDPFlow) {
	protocol := common.GetProtocol(&(flow.Flow))
	mnp.numPackets.AddValue(protocol, len(flow.Packets))
}

// Export returns the metric data per Protocol
func (mnp *MetricNumPackets) Export(protocolKey common.ProtocolKeyType) int {
	return mnp.numPackets.Export(protocolKey)
}

// Export the stored protocols
func (mnp *MetricNumPackets) GetProtocols() []common.Protocol {
	return mnp.numPackets.GetProtocols()
}

// Name of the Metric
func (mnp *MetricNumPackets) Name() string {
	return "NumPackets"
}

// PrintStatistic prints some statistic to the console
func (mnp *MetricNumPackets) PrintStatistic(verbose bool) {
	fmt.Println("Metric Number of Packets:")
	fmt.Print(mnp.numPackets.GetStatistics(verbose))
}
