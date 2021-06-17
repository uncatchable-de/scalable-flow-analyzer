package standard

import (
	"analysis/metrics/common"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/dustin/go-humanize"
)

// ----------------------------------------
// Everything in this file is only used to export the metrics to JSON
// ----------------------------------------

// MetricProtocolExport Interface which must be implemented by the metrics if they shall be included in the exported json file
type MetricProtocolExport interface {
	Export(common.ProtocolKeyType) int
	GetProtocols() []common.Protocol
	Name() string
}

// MetricUnivariateExport Interface which must be implemented by the metrics if they shall be included in the exported json file
type MetricUnivariateExport interface {
	Export(common.ProtocolKeyType) *common.ExportUnivariateFormat
	GetProtocols() []common.Protocol
	Name() string
}

// MetricUnivariateExport Interface which must be implemented by the metrics if they shall be included in the exported json file
type MetricUnivariateClusterExport interface {
	ExportClusters(common.ProtocolKeyType) *common.ExportUnivariateClusterFormat
	GetProtocols() []common.Protocol
	Name() string
}

// MetricBivariateExport Interface which must be implemented by the metrics if they shall be included in the exported json file
type MetricBivariateExport interface {
	ExportBivariate(common.ProtocolKeyType) *common.ExportBivariateFormat
	GetProtocols() []common.Protocol
	Name() string
}

// MetricBivariateClusterExport Interface which must be implemented by the metrics if they shall be included in the exported json file
type MetricBivariateClusterExport interface {
	ExportBivariateClusters(common.ProtocolKeyType) *common.ExportBivariateClusterFormat
	GetProtocols() []common.Protocol
	Name() string
}

type exportFormat struct {
	ProtocolMetrics          map[string]int
	BivariateMetrics         map[string]*common.ExportBivariateFormat         // map[metricname]ExportBivariateFormat
	BivariateClusterMetrics  map[string]*common.ExportBivariateClusterFormat  // map[metricname]ExportBivariateClusterFormat
	UnivariateMetrics        map[string]*common.ExportUnivariateFormat        // map[metricname]ExportUnivariateFormat
	UnivariateClusterMetrics map[string]*common.ExportUnivariateClusterFormat // map[metricname]ExportUnivariateClusterFormat
}

func addMetricDataToExport(export *exportFormat, metric MetricProtocolExport, protocol common.Protocol) {
	metricData := metric.Export(protocol.ProtocolKey)
	metricName := metric.Name()
	export.ProtocolMetrics[metricName] = metricData
}

func addUnivariateMetricDataToExport(export *exportFormat, metric MetricUnivariateExport, protocol common.Protocol) {
	metricData := metric.Export(protocol.ProtocolKey)
	metricName := metric.Name()
	export.UnivariateMetrics[metricName] = metricData
}

func addUnivariateClusterMetricDataToExport(export *exportFormat, metric MetricUnivariateClusterExport, protocol common.Protocol) {
	metricData := metric.ExportClusters(protocol.ProtocolKey)
	metricName := metric.Name()
	export.UnivariateClusterMetrics[metricName] = metricData
}

func addBivariateMetricDataToExport(export *exportFormat, metric MetricBivariateExport, protocol common.Protocol) {
	metricData := metric.ExportBivariate(protocol.ProtocolKey)
	metricName := metric.Name()
	export.BivariateMetrics[metricName] = metricData
}

func addBivariateClusterMetricDataToExport(export *exportFormat, metric MetricBivariateClusterExport, protocol common.Protocol) {
	metricData := metric.ExportBivariateClusters(protocol.ProtocolKey)
	metricName := metric.Name()
	export.BivariateClusterMetrics[metricName] = metricData
}

// Export stores the metric in JSON files in the "directory"
// It will create one file for each protocol.
func (metric *Metric) Export(directory string) {
	var allProtocols = make(map[common.ProtocolKeyType]common.Protocol)

	// Get protocols from all metrics and only add new protocols to list of all protocols
	for _, singleMetric := range metric.allExportedMetricsUnivariate {
		protocols := singleMetric.GetProtocols()
		for _, protocol := range protocols {
			if _, ok := allProtocols[protocol.ProtocolKey]; !ok {
				allProtocols[protocol.ProtocolKey] = protocol
			}
		}
	}

	fmt.Println("Create Metrics for", humanize.Comma(int64(len(allProtocols))), "protocols")
	for _, protocol := range allProtocols {
		export := exportFormat{
			ProtocolMetrics:          make(map[string]int),
			BivariateMetrics:         make(map[string]*common.ExportBivariateFormat),
			BivariateClusterMetrics:  make(map[string]*common.ExportBivariateClusterFormat),
			UnivariateMetrics:        make(map[string]*common.ExportUnivariateFormat),
			UnivariateClusterMetrics: make(map[string]*common.ExportUnivariateClusterFormat),
		}
		for _, singleMetric := range metric.allExportedMetrics {
			addMetricDataToExport(&export, singleMetric, protocol)
		}
		for _, singleMetric := range metric.allExportedMetricsUnivariate {
			addUnivariateMetricDataToExport(&export, singleMetric, protocol)
		}
		for _, singleMetric := range metric.allExportedMetricsUnivariateCluster {
			addUnivariateClusterMetricDataToExport(&export, singleMetric, protocol)
		}
		for _, singleMetric := range metric.allExportedMetricsBivariate {
			addBivariateMetricDataToExport(&export, singleMetric, protocol)
		}
		for _, singleMetric := range metric.allExportedMetricsBivariateCluster {
			addBivariateClusterMetricDataToExport(&export, singleMetric, protocol)
		}
		b, err := json.Marshal(export)
		if err != nil {
			fmt.Println(err.Error())
			panic("Error during marshalling data")
		}

		filename := path.Join(directory, protocol.GetProtocolString()+".json")
		err = ioutil.WriteFile(filename, b, 0644)
		if err != nil {
			fmt.Println(err.Error())
			panic("Could not export json file " + filename)
		}
	}
	fmt.Println("Export successfull")
}
