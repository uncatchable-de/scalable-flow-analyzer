package standard

import (
	"scalable-flow-analyzer/metrics/common"
	"fmt"
)

type MetricNumSessions struct {
	sessions common.IntMetricUnivariate
}

func newMetricNumSessions() *MetricNumSessions {
	var metricNumSessions = MetricNumSessions{}
	metricNumSessions.sessions = common.NewIntMetricUnivariate(1, false)
	return &metricNumSessions
}

func (mns *MetricNumSessions) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	mns.sessions.AddValue(protocol, user.userClusterIndex, len(user.sessions))
}

// Export returns the metric data per Protocol
func (mns *MetricNumSessions) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mns.sessions.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mns *MetricNumSessions) GetProtocols() []common.Protocol {
	return mns.sessions.GetProtocols()
}

// Name of the Metric
func (mns *MetricNumSessions) Name() string {
	return "NumSessions"
}

// PrintStatistic prints some statistic to the console
func (mns *MetricNumSessions) PrintStatistic(verbose bool) {
	fmt.Println("Metric Number of Sessions:")
	fmt.Print(mns.sessions.GetStatistics(verbose))
}
