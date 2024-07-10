package standard

import (
	"scalable-flow-analyzer/flows"
	"scalable-flow-analyzer/metrics/common"
	"fmt"
)

// MetricSize measures the size of the requests and responses.
// Since the metric is calculated in the same way for requests/responses both metrics are combined in this file.
type MetricSize struct {
	request  common.IntMetricBivariate
	response common.IntMetricBivariate
}

func newMetricSize() *MetricSize {
	var metricSize = MetricSize{}
	metricSize.request = common.NewIntMetricBivariate(1, true, false)
	metricSize.response = common.NewIntMetricBivariate(1, true, false)
	return &metricSize
}

type MetricRequestSize struct {
	metricsize *MetricSize
}

func (ms *MetricSize) GetRequest() *MetricRequestSize {
	return &MetricRequestSize{metricsize: ms}
}

func (ms *MetricSize) Name() string {
	return "Size"
}

// Export returns the metric data per Protocol
func (mrs *MetricRequestSize) ExportBivariateClusters(protocolKey common.ProtocolKeyType) *common.ExportBivariateClusterFormat {
	return mrs.metricsize.request.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mrs *MetricRequestSize) GetProtocols() []common.Protocol {
	return mrs.metricsize.request.GetProtocols()
}

// Name of the Metric
func (mrs *MetricRequestSize) Name() string {
	return "RequestSize"
}

type MetricResponseSize struct {
	metricsize *MetricSize
}

func (ms *MetricSize) GetResponse() *MetricResponseSize {
	return &MetricResponseSize{metricsize: ms}
}

// Export returns the metric data per Protocol
func (mrs *MetricResponseSize) ExportBivariateClusters(protocolKey common.ProtocolKeyType) *common.ExportBivariateClusterFormat {
	return mrs.metricsize.response.ExportClusters(protocolKey)
}

// Export the stored protocols
func (mrs *MetricResponseSize) GetProtocols() []common.Protocol {
	return mrs.metricsize.response.GetProtocols()
}

// Name of the Metric
func (mrs *MetricResponseSize) Name() string {
	return "ResponseSize"
}

func (ms *MetricSize) OnFlush(p common.Protocol, flow *flows.Flow, rrp []*common.RequestResponse) {
	for i, reqRes := range rrp {
		var requestSize = 0
		var responseSize = 0
		for _, request := range reqRes.Requests {
			requestSize += int(request.LengthPayload)
		}
		for _, response := range reqRes.Responses {
			responseSize += int(response.LengthPayload)
		}
		ms.request.AddValue(p, reqRes.ClusterIndex, []int{i + 1, requestSize})
		ms.response.AddValue(p, reqRes.ClusterIndex, []int{i + 1, responseSize})
	}
}

// Calc is used to calculate the values without storing them (E.g. for clustering)
func (ms *MetricSize) calc(rrp []*common.RequestResponse) (reqSizes, resSizes []int) {
	reqSizes = make([]int, len(rrp))
	resSizes = make([]int, len(rrp))
	for i, reqRes := range rrp {
		var requestSize = 0
		var responseSize = 0
		for _, request := range reqRes.Requests {
			requestSize += int(request.LengthPayload)
		}
		for _, response := range reqRes.Responses {
			responseSize += int(response.LengthPayload)
		}
		reqSizes[i] = requestSize
		resSizes[i] = responseSize
	}
	return reqSizes, resSizes
}

// PrintStatistic prints some statistic to the console
func (ms *MetricSize) PrintStatistic(verbose bool) {
	fmt.Println("Metric Size Request:")
	fmt.Print(ms.request.GetStatistics(verbose))
	fmt.Println("Metric Size Response:")
	fmt.Print(ms.response.GetStatistics(verbose))
}
