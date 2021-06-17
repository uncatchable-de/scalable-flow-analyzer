package common

import (
	"fmt"
)

// This metric counts the number of reconstructed packets.
// Since it is only used for debugging and cannot use a official hook, it differs in the implementation.
// Do not use this metric as template to create your own metrics!

type MetricReconstructedPacketsSpeed struct {
	speed IntMetricUnivariate
}

func NewMetricReconstructedPacketsSpeed() *MetricReconstructedPacketsSpeed {
	var metricReconstructedPackets = MetricReconstructedPacketsSpeed{}
	metricReconstructedPackets.speed = NewIntMetricUnivariate(1, true)
	return &metricReconstructedPackets
}

func (mrp *MetricReconstructedPacketsSpeed) addReconstructedPacket(p Protocol, speed, size int) {
	mrp.speed.AddValue(p, DefaultClusterIndex, speed)
}

// Export returns the metric data per Protocol
func (mrp *MetricReconstructedPacketsSpeed) Export(protocolKey ProtocolKeyType) *ExportUnivariateFormat {
	return mrp.speed.Export(protocolKey, DefaultClusterIndex)
}

// Export the stored protocols
func (mrp *MetricReconstructedPacketsSpeed) GetProtocols() []Protocol {
	return mrp.speed.GetProtocols()
}

// Name of the Metric
func (mrp *MetricReconstructedPacketsSpeed) Name() string {
	return "ReconstructedPacketsSpeed"
}

// PrintStatistic prints some statistic to the console
func (mrp *MetricReconstructedPacketsSpeed) PrintStatistic(verbose bool) {
	fmt.Println("Metric ReconstructedPackets:")
	fmt.Print(mrp.speed.GetStatistics(verbose))
}
