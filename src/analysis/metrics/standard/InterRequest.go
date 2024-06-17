package standard

import (
	"scalable-flow-analyzer/flows"
	"scalable-flow-analyzer/metrics/common"
	"fmt"
)

type MetricInterRequests struct {
	interRequestTimes common.IntMetricBivariate
}

func newMetricInterRequests() *MetricInterRequests {
	var metricInterRequests = MetricInterRequests{}
	metricInterRequests.interRequestTimes = common.NewIntMetricBivariate(1, true, false)
	return &metricInterRequests
}

// Calc is used to calculate the values without storing them (E.g. for clustering)
func (mir *MetricInterRequests) calc(rrp []*common.RequestResponse) [][]int {
	var interRequestTimes [][]int
	for i := 1; i < len(rrp); i++ {
		var interReqTime = int(rrp[i].Requests[0].Timestamp - rrp[i-1].Requests[0].Timestamp)
		interRequestTimes = append(interRequestTimes, []int{i, interReqTime})
	}
	return interRequestTimes
}

func (mir *MetricInterRequests) OnFlush(p common.Protocol, flow *flows.Flow, rrp []*common.RequestResponse) {
	interRequestTimes := mir.calc(rrp)
	if len(interRequestTimes) > 0 {
		mir.interRequestTimes.AddValue(p, flow.ClusterIndex, interRequestTimes...)
	}
}

// Export returns the metric data per Protocol
func (mir *MetricInterRequests) ExportBivariateClusters(protocolKey common.ProtocolKeyType) *common.ExportBivariateClusterFormat {
	return mir.interRequestTimes.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mir *MetricInterRequests) GetProtocols() []common.Protocol {
	return mir.interRequestTimes.GetProtocols()
}

// Name of the Metric
func (mir *MetricInterRequests) Name() string {
	return "InterRequestTimes"
}

// PrintStatistic prints some statistic to the console
func (mir *MetricInterRequests) PrintStatistic(verbose bool) {
	fmt.Println("Metric Interrequest times:")
	fmt.Print(mir.interRequestTimes.GetStatistics(verbose))
}
