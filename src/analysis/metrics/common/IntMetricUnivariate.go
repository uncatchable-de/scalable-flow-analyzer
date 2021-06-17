package common

import (
	"fmt"
	"sort"
	"sync"
)

type intMetricUnivariateCluster struct {
	values       map[int]int // [value]counter
	clusterIndex int
	mutex        sync.Mutex
}

type intMetricUnivariateProtocol struct {
	clusters map[int]*intMetricUnivariateCluster // map[ClusterIndex]intMetricUnivariateCluster
	protocol Protocol
	mutex    sync.RWMutex
}

type IntMetricUnivariate struct {
	resolution      int
	logScaling      bool
	mutex           sync.RWMutex
	protocolMetrics map[ProtocolKeyType]*intMetricUnivariateProtocol
}

// NewIntMetric creates a new Univariate Integer Metrics
func NewIntMetricUnivariate(resolution int, logScaling bool) IntMetricUnivariate {
	return IntMetricUnivariate{
		resolution:      resolution,
		protocolMetrics: make(map[ProtocolKeyType]*intMetricUnivariateProtocol),
		logScaling:      logScaling}
}

// AddValue adds one or multiple values to the Metric. It will scale all values to the configured resolution
func (imu *IntMetricUnivariate) AddValue(protocol Protocol, clusterIndex int, values ...int) {
	var intMetricProt *intMetricUnivariateProtocol
	var intMetricCluster *intMetricUnivariateCluster
	var ok bool

	// Scale Values
	if imu.logScaling {
		// Scale metric values dynamically
		for i := range values {
			values[i] = scaleToLog(values[i])
		}
	} else {
		// Scale metric values
		for i := range values {
			values[i] = scaleToResolution(values[i], imu.resolution)
		}
	}

	// Add Protocol if not exists
	imu.mutex.RLock()
	intMetricProt, ok = imu.protocolMetrics[protocol.ProtocolKey]
	imu.mutex.RUnlock()
	if !ok {
		// This double checking is necessary to be thread safe.
		// I know it is more complicated, but this allows reads to be done in parallel (was big blocker according to block profiler)
		imu.mutex.Lock()
		if intMetricProt, ok = imu.protocolMetrics[protocol.ProtocolKey]; !ok {
			intMetricProt = &intMetricUnivariateProtocol{protocol: protocol, clusters: make(map[int]*intMetricUnivariateCluster)}
			imu.protocolMetrics[protocol.ProtocolKey] = intMetricProt
		}
		imu.mutex.Unlock()
	}

	// Add Cluster if not exist
	intMetricProt.mutex.RLock()
	intMetricCluster, ok = intMetricProt.clusters[clusterIndex]
	intMetricProt.mutex.RUnlock()
	if !ok {
		// This double checking is necessary to be thread safe.
		// I know it is more complicated, but this allows reads to be done in parallel (was big blocker according to block profiler)
		intMetricProt.mutex.Lock()
		if intMetricCluster, ok = intMetricProt.clusters[clusterIndex]; !ok {
			intMetricCluster = &intMetricUnivariateCluster{clusterIndex: clusterIndex, values: make(map[int]int)}
			intMetricProt.clusters[clusterIndex] = intMetricCluster
		}
		intMetricProt.mutex.Unlock()
	}

	// Write metric values (thread-safe)
	intMetricCluster.mutex.Lock()
	for i := range values {
		intMetricCluster.values[values[i]]++
	}
	intMetricCluster.mutex.Unlock()
}

// Export the metrics for one cluster
func (imu *IntMetricUnivariate) Export(protocolKey ProtocolKeyType, clusterIndex int) *ExportUnivariateFormat {
	export := &ExportUnivariateFormat{Values: make([][]int, 0)}
	valueKeys := make([]int, 0)

	// Check if protocol exists
	imu.mutex.RLock()
	defer imu.mutex.RUnlock()
	if _, ok := imu.protocolMetrics[protocolKey]; !ok {
		return export
	}

	// Check if cluster exists
	imu.protocolMetrics[protocolKey].mutex.RLock()
	defer imu.protocolMetrics[protocolKey].mutex.RUnlock()
	var metricPointer *intMetricUnivariateCluster
	var ok bool
	if metricPointer, ok = imu.protocolMetrics[protocolKey].clusters[clusterIndex]; !ok {
		return export
	}

	metricPointer.mutex.Lock()
	defer metricPointer.mutex.Unlock()

	for value := range metricPointer.values {
		valueKeys = append(valueKeys, value)
	}
	sort.Ints(valueKeys)
	for _, value := range valueKeys {
		export.Values = append(export.Values, []int{value, metricPointer.values[value]})
	}
	return export
}

func (imu *IntMetricUnivariate) ExportClusters(protocolKey ProtocolKeyType) *ExportUnivariateClusterFormat {
	export := &ExportUnivariateClusterFormat{Clusters: make(map[int]*ExportUnivariateFormat)}
	imu.mutex.RLock()
	defer imu.mutex.RUnlock()
	if _, ok := imu.protocolMetrics[protocolKey]; !ok {
		return export
	}
	for clusterIndex := range imu.protocolMetrics[protocolKey].clusters {
		export.Clusters[clusterIndex] = imu.Export(protocolKey, clusterIndex)
	}
	return export
}

// Export the Protocols
func (imu *IntMetricUnivariate) GetProtocols() []Protocol {
	var protocols = make([]Protocol, 0)
	imu.mutex.RLock()
	for _, intMetricProt := range imu.protocolMetrics {
		protocols = append(protocols, intMetricProt.protocol)
	}
	imu.mutex.RUnlock()
	return protocols
}

// GetStatistics returns a string with the most important information. Use fmt.Print() to print it
func (imu *IntMetricUnivariate) GetStatistics(verbose bool) string {
	var numberOfProtocols = len(imu.protocolMetrics)
	statistic := fmt.Sprintln(" Number of Protocols: \t", numberOfProtocols)
	if verbose {
		for _, intMetricProt := range imu.protocolMetrics {
			for _, intMetricClus := range intMetricProt.clusters {
				var meanValues int
				for value, count := range intMetricClus.values {
					meanValues += (value * count)
				}
				statistic += fmt.Sprintln(intMetricProt.protocol.GetProtocolString(), ":", meanValues/len(intMetricClus.values))
			}
		}
	}
	return statistic
}
