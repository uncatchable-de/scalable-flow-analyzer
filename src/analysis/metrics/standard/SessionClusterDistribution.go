package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricSessionClusterDistribution struct {
	clusterDistribution common.IntMetricUnivariate
}

func newMetricSessionClusterDistribution() *MetricSessionClusterDistribution {
	var metricSessionClusterDistribution = MetricSessionClusterDistribution{}
	metricSessionClusterDistribution.clusterDistribution = common.NewIntMetricUnivariate(1, false)
	return &metricSessionClusterDistribution
}

func (mcd *MetricSessionClusterDistribution) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	var clusterDistribution = make([]int, 0)
	for _, session := range user.sessions {
		clusterDistribution = append(clusterDistribution, session.sessionClusterIndex)
	}
	mcd.clusterDistribution.AddValue(protocol, user.userClusterIndex, clusterDistribution...)
}

// Export returns the metric data per Protocol
func (mcd *MetricSessionClusterDistribution) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mcd.clusterDistribution.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mcd *MetricSessionClusterDistribution) GetProtocols() []common.Protocol {
	return mcd.clusterDistribution.GetProtocols()
}

// Name of the Metric
func (mcd *MetricSessionClusterDistribution) Name() string {
	return "SessionClusterDistribution"
}

// PrintStatistic prints some statistic to the console
func (mcd *MetricSessionClusterDistribution) PrintStatistic(verbose bool) {
	fmt.Println("Metric Session Cluster distribution:")
	fmt.Print(mcd.clusterDistribution.GetStatistics(verbose))
}
