package common

import (
	"fmt"
	"sort"
	"sync"
)

type intMetricBivariateCluster struct {
	values       map[int]map[int]int // map[variable]map[value]counter
	clusterIndex int
	mutex        sync.Mutex
}

type intMetricBivariateProtocol struct {
	clusters map[int]*intMetricBivariateCluster // map[clusterIndex]*IntMetricBivariateCluster
	protocol Protocol
	mutex    sync.RWMutex
}

type IntMetricBivariate struct {
	resolution         int
	logScaling         bool // inidcates wether the value shall be logscaled.
	variableLogScaling bool // indicates whether the variable should be logscaled as well
	mutex              sync.RWMutex
	protocolMetrics    map[ProtocolKeyType]*intMetricBivariateProtocol
}

// NewIntMetricBivariate creates a new bivariate Integer Metric
func NewIntMetricBivariate(resolution int, logScaling, variableLogScaling bool) IntMetricBivariate {
	return IntMetricBivariate{
		resolution:      resolution,
		protocolMetrics: make(map[ProtocolKeyType]*intMetricBivariateProtocol),
		logScaling:      logScaling}
}

// AddValue adds one or multiple values to the Metric. It will scale all values to the configured resolution
// It expects that each value is a tuple with [variable, value].
func (imb *IntMetricBivariate) AddValue(protocol Protocol, clusterIndex int, values ...[]int) {
	var intMetricProt *intMetricBivariateProtocol
	var intMetricCluster *intMetricBivariateCluster
	var ok bool

	// Scale Values
	if imb.logScaling || imb.variableLogScaling {
		// Scale metric values dynamically
		for i := range values {
			if len(values[i]) != 2 {
				panic("Add value exepcts a tuple for bivariate metrics: variable, value")
			}
			if imb.variableLogScaling {
				values[i][0] = scaleToLog(values[i][0])
			}
			if imb.logScaling {
				values[i][1] = scaleToLog(values[i][1])
			}
		}
	} else if imb.resolution != 1 {
		// Scale metric values
		for i := range values {
			values[i][1] = scaleToResolution(values[i][1], imb.resolution)
		}
	}

	// Add Protocol if not exists
	imb.mutex.RLock()
	intMetricProt, ok = imb.protocolMetrics[protocol.ProtocolKey]
	imb.mutex.RUnlock()
	if !ok {
		// This double checking is necessary to be thread safe.
		// I know it is more complicated, but this allows reads to be done in parallel (was big blocker according to block profiler)
		imb.mutex.Lock()
		if intMetricProt, ok = imb.protocolMetrics[protocol.ProtocolKey]; !ok {
			intMetricProt = &intMetricBivariateProtocol{protocol: protocol, clusters: make(map[int]*intMetricBivariateCluster)}
			imb.protocolMetrics[protocol.ProtocolKey] = intMetricProt
		}
		imb.mutex.Unlock()
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
			intMetricCluster = &intMetricBivariateCluster{clusterIndex: clusterIndex, values: make(map[int]map[int]int)}
			intMetricProt.clusters[clusterIndex] = intMetricCluster
		}
		intMetricProt.mutex.Unlock()
	}

	// Write metric values (thread-safe)
	intMetricCluster.mutex.Lock()
	for i := range values {
		if _, ok := intMetricCluster.values[values[i][0]]; !ok {
			intMetricCluster.values[values[i][0]] = make(map[int]int)
		}
		intMetricCluster.values[values[i][0]][values[i][1]]++
	}
	intMetricCluster.mutex.Unlock()
}

// Export the metrics for one cluster
func (imb *IntMetricBivariate) Export(protocolKey ProtocolKeyType, clusterIndex int) *ExportBivariateFormat {
	export := &ExportBivariateFormat{Variable: make(map[int]*ExportUnivariateFormat)}

	// Check if protocol exists
	imb.mutex.RLock()
	defer imb.mutex.RUnlock()
	if _, ok := imb.protocolMetrics[protocolKey]; !ok {
		fmt.Println("Protocol does not exist")
		return export
	}

	// Check if cluster exists
	imb.protocolMetrics[protocolKey].mutex.RLock()
	defer imb.protocolMetrics[protocolKey].mutex.RUnlock()
	var metricPointer *intMetricBivariateCluster
	var ok bool
	if metricPointer, ok = imb.protocolMetrics[protocolKey].clusters[clusterIndex]; !ok {
		fmt.Println("cluster does not exist")
		return export
	}

	metricPointer.mutex.Lock()
	defer metricPointer.mutex.Unlock()

	for variableValue := range metricPointer.values {
		export.Variable[variableValue] = &ExportUnivariateFormat{Values: make([][]int, 0)}
		valueKeys := make([]int, 0)

		for value := range metricPointer.values[variableValue] {
			valueKeys = append(valueKeys, value)
		}
		// Sort them by value
		sort.Ints(valueKeys)
		for _, value := range valueKeys {
			var valueCounterTuple = []int{value, metricPointer.values[variableValue][value]}
			export.Variable[variableValue].Values = append(export.Variable[variableValue].Values, valueCounterTuple)
		}
	}
	return export
}

func (imb *IntMetricBivariate) ExportClusters(protocolKey ProtocolKeyType) *ExportBivariateClusterFormat {
	export := &ExportBivariateClusterFormat{Clusters: make(map[int]*ExportBivariateFormat)}
	imb.mutex.RLock()
	defer imb.mutex.RUnlock()
	if _, ok := imb.protocolMetrics[protocolKey]; !ok {
		return export
	}
	for clusterIndex := range imb.protocolMetrics[protocolKey].clusters {
		export.Clusters[clusterIndex] = imb.Export(protocolKey, clusterIndex)
	}
	return export
}

// Export the Protocols
func (imb *IntMetricBivariate) GetProtocols() []Protocol {
	var protocols = make([]Protocol, 0)
	imb.mutex.RLock()
	for _, intMetricProt := range imb.protocolMetrics {
		protocols = append(protocols, intMetricProt.protocol)
	}
	imb.mutex.RUnlock()
	return protocols
}

// GetStatistics returns a string with the most important informations. Use fmt.Print() to print it
func (imb *IntMetricBivariate) GetStatistics(verbose bool) string {
	var numberOfProtocols = len(imb.protocolMetrics)
	statistic := fmt.Sprintln(" Number of Protocols: \t", numberOfProtocols)
	return statistic
}
