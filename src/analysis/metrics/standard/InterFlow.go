package standard

import (
	"analysis/metrics/common"
	"fmt"
)

type MetricInterFlow struct {
	interFlowTimes common.IntMetricUnivariate
}

func newMetricInterFlows() *MetricInterFlow {
	var metricInterFlows = MetricInterFlow{}
	metricInterFlows.interFlowTimes = common.NewIntMetricUnivariate(1, true)
	return &metricInterFlows
}

func (mif *MetricInterFlow) calc(session *session) []int {
	var interFlowTimes []int
	for i := 1; i < len(session.flows); i++ {
		interFlowTimes = append(interFlowTimes, int(session.flows[i].start-session.flows[i-1].start))
	}
	return interFlowTimes
}

func (mif *MetricInterFlow) OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct) {
	for _, session := range user.sessions {
		interFlowTimes := mif.calc(session)
		if len(interFlowTimes) > 0 {
			mif.interFlowTimes.AddValue(protocol, session.sessionClusterIndex, interFlowTimes...)
		}
	}
}

// Export returns the metric data per Protocol
func (mif *MetricInterFlow) ExportClusters(protocolKey common.ProtocolKeyType) *common.ExportUnivariateClusterFormat {
	return mif.interFlowTimes.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mif *MetricInterFlow) GetProtocols() []common.Protocol {
	return mif.interFlowTimes.GetProtocols()
}

// Name of the Metric
func (mif *MetricInterFlow) Name() string {
	return "InterFlowTimes"
}

// PrintStatistic prints some statistic to the console
func (mif *MetricInterFlow) PrintStatistic(verbose bool) {
	fmt.Println("Metric InterFlowTimes:")
	fmt.Print(mif.interFlowTimes.GetStatistics(verbose))
}
