package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricNumFlows struct {
	flows common.IntMetricUnivariate
}

func newMetricNumFlows() *MetricNumFlows {
	var metricNumFlows = MetricNumFlows{}
	metricNumFlows.flows = common.NewIntMetricUnivariate(1, false)
	return &metricNumFlows
}

func (mnf *MetricNumFlows) calc(session *session) int {
	return len(session.flows)
}

func (mnf *MetricNumFlows) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	for _, session := range user.sessions {
		mnf.flows.AddValue(protocol, session.sessionClusterIndex, mnf.calc(session))
	}
}

// Export returns the metric data per Protocol
func (mnf *MetricNumFlows) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mnf.flows.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mnf *MetricNumFlows) GetProtocols() []common.Protocol {
	return mnf.flows.GetProtocols()
}

// Name of the Metric
func (mnf *MetricNumFlows) Name() string {
	return "NumFlows"
}

// PrintStatistic prints some statistic to the console
func (mnf *MetricNumFlows) PrintStatistic(verbose bool) {
	fmt.Println("Metric Number of Flows:")
	fmt.Print(mnf.flows.GetStatistics(verbose))
}
