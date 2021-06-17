package standard

// This package contains all Metrics.
// There exist several kind of Metrics, identified by their layer (Flow, RequestResponse (RR), Session).
// The Pool will only flush Flows (connections). Based on these, sessions and request/responses are extracted by this package.

import (
	"analysis/flows"
	"analysis/metrics/common"
	"sync"
)

// Metric handles all sub-metrics.
// It must be created with the NewMetric function, to ensure all sessionMetrics and RRMetrics are registered correctly.
// The caller must register the metric by the pool.
// At the end of the processing, you must call ForceFlush to flush all remaining sessions.
type Metric struct {
	ReqResIdentifier                 *common.ReqResIdentifier
	SessionIdentifier                *sessionIdentifier
	MetricSize                       *MetricSize
	MetricInterRequest               *MetricInterRequests
	MetricNumRRPairs                 *MetricNumRRPairs
	MetricNumSessions                *MetricNumSessions
	MetricInterSessions              *MetricInterSessions
	MetricNumFlows                   *MetricNumFlows
	MetricFlowRate                   *MetricFlowRate
	MetricInterFlowTimes             *MetricInterFlow
	MetricNumPackets                 *MetricNumPackets
	MetricNumServers                 *MetricNumServers
	MetricRRPClusterDistribution     *MetricRRPClusterDistribution
	MetricFlowClusterDistribution    *MetricFlowClusterDistribution
	MetricSessionClusterDistribution *MetricSessionClusterDistribution
	MetricUserClusterDistribution    *MetricUserClusterDistribution
	registeredRRMetrics              []RRMetric
	registeredFlowMetrics            []FlowMetric
	// The following slices are used to export metrics automatically
	allExportedMetrics                  []MetricProtocolExport
	allExportedMetricsUnivariate        []MetricUnivariateExport
	allExportedMetricsUnivariateCluster []MetricUnivariateClusterExport
	allExportedMetricsBivariate         []MetricBivariateExport
	allExportedMetricsBivariateCluster  []MetricBivariateClusterExport

	clusterController *ClusterController
}

// FlowMetric are metrics which are evaluated on flow level.
type FlowMetric interface {
	OnTCPFlush(flow *flows.TCPFlow)
	OnUDPFlush(flow *flows.UDPFlow)
	PrintStatistic(verbose bool)
}

// RRMetric are metrics which are evaluated on request response level.
type RRMetric interface {
	OnFlush(p common.Protocol, flow *flows.Flow, reqRes []*common.RequestResponse)
	PrintStatistic(verbose bool)
	Name() string
}

// SessionMetric are metrics which are evaluated on session level.
type SessionMetric interface {
	OnFlush(protocol common.Protocol, userAddress uint64, user *userSessionsStruct)
	PrintStatistic(verbose bool)
}

// NewMetric creates a new Metric and registers all session and request/response metrics
// If infoPath is not empty, flow and session information will be stored to this directory
// if clusterModelDirectory is not empty, a clustering will be used.
func NewMetric(sessionTimeout int64, infoPath, clusterModelDirectory string,
	dropUnidirectionalFlows, reconstructTCPResponse, statisticTCPReconstruction bool) *Metric {
	var metric = &Metric{}
	metric.clusterController = NewClusterController(metric, infoPath, clusterModelDirectory)

	// Session and Request/Response Identifier
	metric.SessionIdentifier = newSessionIdentifier(sessionTimeout, metric.clusterController)
	metric.registerFlowMetric(metric.SessionIdentifier)

	var reconstructionMetricSpeed *common.MetricReconstructedPacketsSpeed
	var reconstructionMetricSize *common.MetricReconstructedPacketsSize
	if statisticTCPReconstruction {
		// The reconstruction metric is automatically hooked at request response metric.
		// We only must add it to the export
		reconstructionMetricSpeed = common.NewMetricReconstructedPacketsSpeed()
		metric.allExportedMetricsUnivariate = append(metric.allExportedMetricsUnivariate, reconstructionMetricSpeed)
		// The reconstruction metric is automatically hooked at request response metric.
		// We only must add it to the export
		reconstructionMetricSize = common.NewMetricReconstructedPacketsSize()
		metric.allExportedMetricsUnivariate = append(metric.allExportedMetricsUnivariate, reconstructionMetricSize)
	}
	metric.ReqResIdentifier = common.NewReqResIdentifier(dropUnidirectionalFlows, reconstructTCPResponse,
		reconstructionMetricSpeed, reconstructionMetricSize)

	// Flow Metrics
	metric.MetricNumPackets = newMetricNumPackets()
	metric.registerFlowMetric(metric.MetricNumPackets)
	metric.allExportedMetrics = append(metric.allExportedMetrics, metric.MetricNumPackets)

	metric.MetricFlowRate = newMetricFlowRate()
	metric.registerFlowMetric(metric.MetricFlowRate)
	metric.allExportedMetricsUnivariate = append(metric.allExportedMetricsUnivariate, metric.MetricFlowRate)

	// RequestResponse Metrics
	metric.MetricRRPClusterDistribution = newMetricRRPClusterDistribution()
	metric.registerRRMetric(metric.MetricRRPClusterDistribution)
	metric.allExportedMetricsBivariateCluster = append(metric.allExportedMetricsBivariateCluster, metric.MetricRRPClusterDistribution)

	metric.MetricSize = newMetricSize()
	metric.registerRRMetric(metric.MetricSize)
	metric.allExportedMetricsBivariateCluster = append(metric.allExportedMetricsBivariateCluster,
		metric.MetricSize.GetRequest(), metric.MetricSize.GetResponse())

	metric.MetricInterRequest = newMetricInterRequests()
	metric.registerRRMetric(metric.MetricInterRequest)
	metric.allExportedMetricsBivariateCluster = append(metric.allExportedMetricsBivariateCluster, metric.MetricInterRequest)

	metric.MetricNumRRPairs = newMetricNumRRPairs()
	metric.registerRRMetric(metric.MetricNumRRPairs)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricNumRRPairs)

	// Session metrics
	metric.MetricNumSessions = newMetricNumSessions()
	metric.registerSessionMetric(metric.MetricNumSessions)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricNumSessions)

	metric.MetricInterSessions = newMetricInterSessions()
	metric.registerSessionMetric(metric.MetricInterSessions)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricInterSessions)

	metric.MetricNumFlows = newMetricNumFlows()
	metric.registerSessionMetric(metric.MetricNumFlows)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricNumFlows)

	metric.MetricInterFlowTimes = newMetricInterFlows()
	metric.registerSessionMetric(metric.MetricInterFlowTimes)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricInterFlowTimes)

	metric.MetricNumServers = newMetricNumServers()
	metric.registerSessionMetric(metric.MetricNumServers)
	metric.allExportedMetricsBivariateCluster = append(metric.allExportedMetricsBivariateCluster, metric.MetricNumServers)

	metric.MetricFlowClusterDistribution = newMetricFlowClusterDistribution()
	metric.registerSessionMetric(metric.MetricFlowClusterDistribution)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricFlowClusterDistribution)

	metric.MetricSessionClusterDistribution = newMetricSessionClusterDistribution()
	metric.registerSessionMetric(metric.MetricSessionClusterDistribution)
	metric.allExportedMetricsUnivariateCluster = append(metric.allExportedMetricsUnivariateCluster, metric.MetricSessionClusterDistribution)

	metric.MetricUserClusterDistribution = newMetricUserClusterDistribution()
	metric.registerSessionMetric(metric.MetricUserClusterDistribution)
	metric.allExportedMetricsUnivariate = append(metric.allExportedMetricsUnivariate, metric.MetricUserClusterDistribution)
	return metric
}

func (metric *Metric) registerRRMetric(rrMetric RRMetric) {
	metric.registeredRRMetrics = append(metric.registeredRRMetrics, rrMetric)
}

func (metric *Metric) registerFlowMetric(flowMetric FlowMetric) {
	metric.registeredFlowMetrics = append(metric.registeredFlowMetrics, flowMetric)
}

func (metric *Metric) registerSessionMetric(sessionMetric SessionMetric) {
	metric.SessionIdentifier.registerSessionMetric(sessionMetric)
}

// OnTCPFlush we first identify the request/response pairs. Based on these,
// the basic metrics to identify the corresponding cluster can be calculated.
// Afterwards, all metrics are computed.
// Session Metrics are called by sessionIdentifier on ForceFlush
func (metric *Metric) OnTCPFlush(flow *flows.TCPFlow) {
	var protocol = common.GetProtocol(&flow.Flow)
	reqRes, dropFlow := metric.ReqResIdentifier.OnTCPFlush(protocol, flow)
	if dropFlow {
		return
	}

	metric.clusterController.CollectAndSetFlowClusterIndex(&flow.Flow, reqRes)
	metric.clusterController.CollectAndSetRRPClusterIndex(&flow.Flow, reqRes)

	for _, metric := range metric.registeredRRMetrics {
		metric.OnFlush(protocol, &flow.Flow, reqRes)
	}

	for _, metric := range metric.registeredFlowMetrics {
		metric.OnTCPFlush(flow)
	}
}

// OnUDPFlush we first identify the request/response pairs. Based on these,
// the basic metrics to identify the corresponding cluster can be calculated.
// Afterwards, all metrics are computed.
// Session Metrics are called by sessionIdentifier on ForceFlush
func (metric *Metric) OnUDPFlush(flow *flows.UDPFlow) {
	var protocol = common.GetProtocol(&flow.Flow)
	reqRes, dropFlow := metric.ReqResIdentifier.OnUDPFlush(protocol, flow)
	if dropFlow {
		return
	}

	metric.clusterController.CollectAndSetRRPClusterIndex(&flow.Flow, reqRes)
	metric.clusterController.CollectAndSetFlowClusterIndex(&flow.Flow, reqRes)

	for _, metric := range metric.registeredRRMetrics {
		metric.OnFlush(protocol, &flow.Flow, reqRes)
	}

	for _, metric := range metric.registeredFlowMetrics {
		metric.OnUDPFlush(flow)
	}
}

// ForceFlush flushes all open sessions, so that session metrics also process the remaining sessions
func (metric *Metric) ForceFlush() {
	metric.SessionIdentifier.forceFlush()
	// Save infos in file, which can then be used to calculate clusters
	// Do this in parallel
	var wgPersistInfo sync.WaitGroup
	wgPersistInfo.Add(4)
	go func() {
		metric.clusterController.PersistSessionInfos(true)
		wgPersistInfo.Done()
	}()
	go func() {
		metric.clusterController.PersistUserInfos(true)
		wgPersistInfo.Done()
	}()
	go func() {
		metric.clusterController.PersistFlowInfos(true)
		wgPersistInfo.Done()
	}()
	go func() {
		metric.clusterController.PersistRRPInfos(true)
		wgPersistInfo.Done()
	}()
	wgPersistInfo.Wait()
}
