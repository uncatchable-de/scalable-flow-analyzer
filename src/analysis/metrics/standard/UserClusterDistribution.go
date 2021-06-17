package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricUserClusterDistribution struct {
	clusterDistribution common.IntMetricUnivariate
}

func newMetricUserClusterDistribution() *MetricUserClusterDistribution {
	var metricUserClusterDistribution = MetricUserClusterDistribution{}
	metricUserClusterDistribution.clusterDistribution = common.NewIntMetricUnivariate(1, false)
	return &metricUserClusterDistribution
}

func (mcd *MetricUserClusterDistribution) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	mcd.clusterDistribution.AddValue(protocol, DefaultClusterIndex, user.userClusterIndex)
}

// Export returns the metric data per Protocol
func (mcd *MetricUserClusterDistribution) Export(protocolKey common.ProtocolKeyType) *common.ExportUnivariateFormat {
	return mcd.clusterDistribution.Export(protocolKey, DefaultClusterIndex)
}

// Export the stored protocols
func (mcd *MetricUserClusterDistribution) GetProtocols() []common.Protocol {
	return mcd.clusterDistribution.GetProtocols()
}

// Name of the Metric
func (mcd *MetricUserClusterDistribution) Name() string {
	return "UserClusterDistribution"
}

// PrintStatistic prints some statistic to the console
func (mcd *MetricUserClusterDistribution) PrintStatistic(verbose bool) {
	fmt.Println("Metric User Cluster distribution:")
	fmt.Print(mcd.clusterDistribution.GetStatistics(verbose))
}
