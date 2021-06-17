package standard

import (
	"analysis/flows"
	"analysis/metrics/common"
	"fmt"
)

type MetricRRPClusterDistribution struct {
	clusterDistribution common.IntMetricBivariate
}

func newMetricRRPClusterDistribution() *MetricRRPClusterDistribution {
	var metricClusterDistribution = MetricRRPClusterDistribution{}
	metricClusterDistribution.clusterDistribution = common.NewIntMetricBivariate(1, false, false)
	return &metricClusterDistribution
}

func (mcd *MetricRRPClusterDistribution) OnFlush(p common.Protocol, flow *flows.Flow, rrps []*common.RequestResponse) {
	var clusterDistribution = make([][]int, 0)
	for i, rrp := range rrps {
		clusterDistribution = append(clusterDistribution, []int{i, rrp.ClusterIndex})
	}
	mcd.clusterDistribution.AddValue(p, flow.ClusterIndex, clusterDistribution...)
}

// Export returns the metric data per Protocol
func (mcd *MetricRRPClusterDistribution) ExportBivariateClusters(protocolKey common.ProtocolKeyType) *common.ExportBivariateClusterFormat {
	return mcd.clusterDistribution.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mcd *MetricRRPClusterDistribution) GetProtocols() []common.Protocol {
	return mcd.clusterDistribution.GetProtocols()
}

// Name of the Metric
func (mcd *MetricRRPClusterDistribution) Name() string {
	return "RRPClusterDistribution"
}

// PrintStatistic prints some statistic to the console
func (mcd *MetricRRPClusterDistribution) PrintStatistic(verbose bool) {
	fmt.Println("Metric RRP Cluster distribution:")
	fmt.Print(mcd.clusterDistribution.GetStatistics(verbose))
}
