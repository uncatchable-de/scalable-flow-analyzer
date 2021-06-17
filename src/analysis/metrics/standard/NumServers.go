package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricNumServers struct {
	numServers common.IntMetricBivariate
}

func newMetricNumServers() *MetricNumServers {
	var metricNumServer = MetricNumServers{}
	metricNumServer.numServers = common.NewIntMetricBivariate(1, false, false)
	return &metricNumServer
}

func (mns *MetricNumServers) calc(session *session) int {
	servers := make(map[uint64]bool)
	for _, flow := range session.flows {
		servers[flow.serverAddr] = true
	}
	return len(servers)
}

func (mns *MetricNumServers) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	for _, session := range user.sessions {
		numServers := []int{len(session.flows), mns.calc(session)}
		mns.numServers.AddValue(protocol, session.sessionClusterIndex, numServers)
	}
}

// Export returns the metric data per Protocol
func (mns *MetricNumServers) ExportBivariateClusters(protocolKey common.ProtocolKeyType) *common.ExportBivariateClusterFormat {
	return mns.numServers.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mns *MetricNumServers) GetProtocols() []common.Protocol {
	return mns.numServers.GetProtocols()
}

// Name of the Metric
func (mns *MetricNumServers) Name() string {
	return "NumServers"
}

// PrintStatistic prints some statistic to the console
func (mns *MetricNumServers) PrintStatistic(verbose bool) {
	fmt.Println("Metric Number of Servers:")
	fmt.Print(mns.numServers.GetStatistics(verbose))
}
