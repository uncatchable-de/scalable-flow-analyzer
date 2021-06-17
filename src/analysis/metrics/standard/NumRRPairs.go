package standard

import (
	"analysis/flows"
	"analysis/metrics/common"
	"fmt"
)

type MetricNumRRPairs struct {
	rrPairs common.IntMetricUnivariate
}

func newMetricNumRRPairs() *MetricNumRRPairs {
	var metricNumRRPairs = MetricNumRRPairs{}
	metricNumRRPairs.rrPairs = common.NewIntMetricUnivariate(1, false)
	return &metricNumRRPairs
}

func (mnrrp *MetricNumRRPairs) OnFlush(p common.Protocol, flow *flows.Flow, rrp []*common.RequestResponse) {
	mnrrp.rrPairs.AddValue(p, flow.ClusterIndex, len(rrp))
}

// Export returns the metric data per Protocol
func (mnrrp *MetricNumRRPairs) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mnrrp.rrPairs.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mnrrp *MetricNumRRPairs) GetProtocols() []common.Protocol {
	return mnrrp.rrPairs.GetProtocols()
}

// Name of the Metric
func (mnrrp *MetricNumRRPairs) Name() string {
	return "NumRRPairs"
}

// PrintStatistic prints some statistic to the console
func (mnrrp *MetricNumRRPairs) PrintStatistic(verbose bool) {
	fmt.Println("Metric Number of RR Pairs:")
	fmt.Print(mnrrp.rrPairs.GetStatistics(verbose))
}
