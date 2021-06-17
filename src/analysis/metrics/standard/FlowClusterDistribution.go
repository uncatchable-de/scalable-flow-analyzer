package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricFlowClusterDistribution struct {
	clusterDistribution common.IntMetricUnivariate
}

func newMetricFlowClusterDistribution() *MetricFlowClusterDistribution {
	var metricClusterDistribution = MetricFlowClusterDistribution{}
	metricClusterDistribution.clusterDistribution = common.NewIntMetricUnivariate(1, false)
	return &metricClusterDistribution
}

func (mcd *MetricFlowClusterDistribution) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	for _, session := range user.sessions {
		var clusterDistribution = make([]int, 0)
		for _, flow := range session.flows {
			clusterDistribution = append(clusterDistribution, flow.clusterIndex)
		}
		mcd.clusterDistribution.AddValue(protocol, session.sessionClusterIndex, clusterDistribution...)
	}
}

// Export returns the metric data per Protocol
func (mcd *MetricFlowClusterDistribution) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mcd.clusterDistribution.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mcd *MetricFlowClusterDistribution) GetProtocols() []common.Protocol {
	return mcd.clusterDistribution.GetProtocols()
}

// Name of the Metric
func (mcd *MetricFlowClusterDistribution) Name() string {
	return "FlowClusterDistribution"
}

// PrintStatistic prints some statistic to the console
func (mcd *MetricFlowClusterDistribution) PrintStatistic(verbose bool) {
	fmt.Println("Metric Flow Cluster distribution:")
	fmt.Print(mcd.clusterDistribution.GetStatistics(verbose))
}
