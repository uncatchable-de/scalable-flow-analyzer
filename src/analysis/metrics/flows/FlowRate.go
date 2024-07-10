package flows

import (
	"scalable-flow-analyzer/flows"
	"time"
)

type MetricFlowRate struct {
	// Sampling rate in milliseconds. If equal to zero the
	// average over the entirety of the flow will be calculated.
	samplingRate int64
}

func newMetricFlowRate() *MetricFlowRate {
	return &MetricFlowRate{}
}

func (mfr *MetricFlowRate) calc(flow *flows.Flow) ValueFlowRate {
	var flowRates []uint
	var flowRatesClient []uint
	var flowRatesServer []uint

	size := 0
	sizeClient := 0
	sizeServer := 0

	packets := flow.Packets

	sampleTimespan := time.Millisecond.Nanoseconds() * mfr.samplingRate
	sampleSeconds := float64(sampleTimespan) / float64(time.Second.Nanoseconds())

	start := packets[0].Timestamp
	nextSampleStart := start + sampleTimespan
	for i := 0; i < len(packets); i++ {
		p := packets[i]
		payloadLength := int(p.LengthPayload)

		if p.Timestamp >= nextSampleStart {
			nextSampleStart += sampleTimespan

			flowRates = append(flowRates, uint(float64(size)/sampleSeconds))
			flowRatesClient = append(flowRatesClient, uint(float64(sizeClient)/sampleSeconds))
			flowRatesServer = append(flowRatesServer, uint(float64(sizeServer)/sampleSeconds))

			size = payloadLength
			sizeClient = 0
			sizeServer = 0

			if p.FromClient {
				sizeClient = payloadLength
			} else {
				sizeServer = payloadLength
			}
		} else {
			size += payloadLength

			if p.FromClient {
				sizeClient += payloadLength
			} else {
				sizeServer += payloadLength
			}
		}
	}

	flowRates = append(flowRates, uint(float64(size)/sampleSeconds))
	flowRatesClient = append(flowRatesClient, uint(float64(sizeClient)/sampleSeconds))
	flowRatesServer = append(flowRatesServer, uint(float64(sizeServer)/sampleSeconds))

	return ValueFlowRate{
		flowRates:       flowRates,
		flowRatesClient: flowRatesClient,
		flowRatesServer: flowRatesServer,
	}
}

func (mfr *MetricFlowRate) calcAverage(flow *flows.Flow) ValueFlowRate {
	var flowRates []uint
	var flowRatesClient []uint
	var flowRatesServer []uint

	size := 0
	sizeClient := 0
	sizeServer := 0

	packets := flow.Packets
	for i := 0; i < len(packets); i++ {
		p := packets[i]
		payloadLength := int(p.LengthPayload)

		size += payloadLength
		if p.FromClient {
			sizeClient += payloadLength
		} else {
			sizeServer += payloadLength
		}
	}

	start := packets[0].Timestamp
	end := packets[len(packets)-1].Timestamp

	seconds := float64(end-start) / float64(time.Second.Nanoseconds())
	if seconds == 0 {
		seconds = 1
	}

	flowRates = append(flowRates, uint(float64(size)/seconds))
	flowRatesClient = append(flowRatesClient, uint(float64(sizeClient)/seconds))
	flowRatesServer = append(flowRatesServer, uint(float64(sizeServer)/seconds))

	return ValueFlowRate{
		flowRates:       flowRates,
		flowRatesClient: flowRatesClient,
		flowRatesServer: flowRatesServer,
	}
}

func (mfr *MetricFlowRate) onFlush(flow *flows.Flow) ExportableValue {
	var value ValueFlowRate
	if mfr.samplingRate == 0 {
		value = mfr.calcAverage(flow)
	} else {
		value = mfr.calc(flow)
	}

	return value
}

type ValueFlowRate struct {
	// All rates observed in a flow.
	flowRates []uint
	// All upstream rates observed in a flow.
	flowRatesClient []uint
	// All downstream rates observed in a flow.
	flowRatesServer []uint
}

func (vfr ValueFlowRate) export() map[string]interface{} {
	return map[string]interface{}{
		"flowRates":       vfr.flowRates,
		"flowRatesClient": vfr.flowRatesClient,
		"flowRatesServer": vfr.flowRatesServer,
	}
}
