package common

import (
	"fmt"
)

// This metric counts the number of reconstructed packets.
// Since it is only used for debugging and cannot use a official hook, it differs in the implementation.
// Do not use this metric as template to create your own metrics!

// DefaultClusterIndex is used whenever a metric does not support clustering, or no clustering is used
const DefaultClusterIndex = 0

type MetricReconstructedPacketsSize struct {
	size IntMetricUnivariate
}

func NewMetricReconstructedPacketsSize() *MetricReconstructedPacketsSize {
	var metricReconstructedPackets = MetricReconstructedPacketsSize{}
	metricReconstructedPackets.size = NewIntMetricUnivariate(1, true)
	return &metricReconstructedPackets
}

func (mrp *MetricReconstructedPacketsSize) addReconstructedPacket(p Protocol, speed, size int) {
	mrp.size.AddValue(p, DefaultClusterIndex, size)
}

// Export returns the metric data per Protocol
func (mrp *MetricReconstructedPacketsSize) Export(protocolKey ProtocolKeyType) *ExportUnivariateFormat {
	return mrp.size.Export(protocolKey, DefaultClusterIndex)
}

// Export the stored protocols
func (mrp *MetricReconstructedPacketsSize) GetProtocols() []Protocol {
	return mrp.size.GetProtocols()
}

// Name of the Metric
func (mrp *MetricReconstructedPacketsSize) Name() string {
	return "ReconstructedPacketsSize"
}

// PrintStatistic prints some statistic to the console
func (mrp *MetricReconstructedPacketsSize) PrintStatistic(verbose bool) {
	fmt.Println("Metric ReconstructedPackets:")
	fmt.Print(mrp.size.GetStatistics(verbose))
}
