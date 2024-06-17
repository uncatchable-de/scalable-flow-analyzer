package standard

import (
	"scalable-flow-analyzer/metrics/common"
	"fmt"
)

type MetricInterSessions struct {
	interSessions common.IntMetricUnivariate
}

func newMetricInterSessions() *MetricInterSessions {
	var metricInterSessions = MetricInterSessions{}
	metricInterSessions.interSessions = common.NewIntMetricUnivariate(1, true)
	return &metricInterSessions
}

func (mis *MetricInterSessions) calc(user *userSessionsStruct) []int {
	var interSessionTimes []int
	for i := 1; i < len(user.sessions); i++ {
		interSessionTimes = append(interSessionTimes, int(user.sessions[i].start-user.sessions[i-1].end))
	}
	return interSessionTimes
}

func (mis *MetricInterSessions) OnFlush(protocol common.Protocol, userAddres uint64, user *userSessionsStruct) {
	interSessionTimes := mis.calc(user)
	if len(interSessionTimes) > 0 {
		mis.interSessions.AddValue(protocol, user.userClusterIndex, interSessionTimes...)
	}
}

// Export returns the metric data per Protocol
func (mis *MetricInterSessions) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mis.interSessions.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mis *MetricInterSessions) GetProtocols() []common.Protocol {
	return mis.interSessions.GetProtocols()
}

// Name of the Metric
func (mis *MetricInterSessions) Name() string {
	return "InterSessionTimes"
}

// PrintStatistic prints some statistic to the console
func (mis *MetricInterSessions) PrintStatistic(verbose bool) {
	fmt.Println("Metric Inter Session times:")
	fmt.Print(mis.interSessions.GetStatistics(verbose))
}
