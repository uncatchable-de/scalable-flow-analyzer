package common

import (
	"fmt"
	"math"
	"sync"

	"github.com/dustin/go-humanize"
)

type intMetricProtocol struct {
	value    int
	protocol Protocol
	mutex    sync.Mutex
}

type IntMetric struct {
	mutex           sync.RWMutex
	protocolMetrics map[ProtocolKeyType]*intMetricProtocol
}

// NewIntMetric creates a new Integer Metrics
func NewIntMetric() IntMetric {
	return IntMetric{
		protocolMetrics: make(map[ProtocolKeyType]*intMetricProtocol)}
}

// AddValue adds one or multiple values to the Metric. It will scale all values to the configured resolution
func (im *IntMetric) AddValue(protocol Protocol, values ...int) {
	var intMetricProt *intMetricProtocol
	var ok bool

	im.mutex.RLock()
	intMetricProt, ok = im.protocolMetrics[protocol.ProtocolKey]
	im.mutex.RUnlock()
	if !ok {
		// This double checking is necessary to be thread safe.
		// I know it is more complicated, but this allows reads to be done in parallel (was big blocker according to block profiler)
		im.mutex.Lock()
		if intMetricProt, ok = im.protocolMetrics[protocol.ProtocolKey]; !ok {
			intMetricProt = &intMetricProtocol{protocol: protocol, value: 0}
			im.protocolMetrics[protocol.ProtocolKey] = intMetricProt
		}
		im.mutex.Unlock()
	}
	// Write metric values (thread-safe)
	intMetricProt.mutex.Lock()
	for _, value := range values {
		intMetricProt.value += value
	}
	intMetricProt.mutex.Unlock()
}

// Export the metrics as (value, count) tuples
func (im *IntMetric) Export(protocolKey ProtocolKeyType) int {
	exportValue := 0
	im.mutex.RLock()
	if _, ok := im.protocolMetrics[protocolKey]; !ok {
		im.mutex.RUnlock()
		return 0
	}
	im.protocolMetrics[protocolKey].mutex.Lock()
	exportValue = im.protocolMetrics[protocolKey].value
	im.protocolMetrics[protocolKey].mutex.Unlock()
	im.mutex.RUnlock()
	return exportValue
}

// Export the Protocols
func (im *IntMetric) GetProtocols() []Protocol {
	var protocols = make([]Protocol, 0)
	im.mutex.RLock()
	for _, intMetricProt := range im.protocolMetrics {
		protocols = append(protocols, intMetricProt.protocol)
	}
	im.mutex.RUnlock()
	return protocols
}

// GetStatistics returns a string with the most important informations. Use fmt.Print() to print it
func (im *IntMetric) GetStatistics(verbose bool) string {
	var numberOfProtocols = len(im.protocolMetrics)
	statistic := fmt.Sprintln(" Number of Protocols: \t", numberOfProtocols)
	if verbose {
		for _, intMetricProt := range im.protocolMetrics {
			statistic += fmt.Sprintln(intMetricProt.protocol.GetProtocolString(), ":", humanize.Comma(int64(intMetricProt.value)))
		}
	}
	return statistic
}

func scaleToResolution(value, resolution int) int {
	value -= (value % resolution)
	return value
}

const scaleFactor = 0.005 // the smaller the scale factor, the more values we will have for each log interval
const upscaleFactor = 1 / scaleFactor

// Playground Code to test different scaleFactors:
// lastScaled := 0
// count := 0
// for i := 0; i < 4000000; i++  {
// 	scaled := scaleToLog(i)
// 	if lastScaled != scaled {
// 		fmt.Println(i, scaled)
// 		count++
// 		lastScaled = scaled
// 	}
// }
// fmt.Println(count)

func scaleToLog(value int) int {
	if value == 0 {
		return 0
	}
	var signNegative = value < 0
	if signNegative {
		value *= -1
	}
	var tmp = math.Log10(float64(value))
	tmp = scaleFactor * math.Floor(tmp*upscaleFactor) // Same as x -(x mod scaleFactor)
	value = int(math.Ceil(math.Pow(10, tmp)))         // Use math.Ceil for precision
	if signNegative {
		value *= -1
	}
	return value
}
