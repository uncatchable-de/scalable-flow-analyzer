package flows

import (
	"analysis/flows"
)

type MetricFlowSize struct{}

func newMetricFlowSize() *MetricFlowSize {
	return &MetricFlowSize{}
}

func (mfs *MetricFlowSize) calc(flow *flows.Flow) (value ValueFlowSize) {
	var size, sizeClient, sizeServer uint

	packets := flow.Packets
	for i := 0; i < len(packets); i++ {
		p := packets[i]
		payloadLength := uint(p.LengthPayload)

		size += payloadLength
		if p.FromClient {
			sizeClient += payloadLength
		} else {
			sizeServer += payloadLength
		}
	}

	return ValueFlowSize{
		size:       size,
		sizeClient: sizeClient,
		sizeServer: sizeServer,
	}
}

func (mfs *MetricFlowSize) onFlush(flow *flows.Flow) ExportableValue {
	value := mfs.calc(flow)
	return value
}

type ValueFlowSize struct {
	// The number of bytes transferred in both directions.
	size uint
	// The number of bytes transferred to the server.
	sizeClient uint
	// The number of bytes transferred to the client.
	sizeServer uint
}

func (vfs ValueFlowSize) export() map[string]interface{} {
	return map[string]interface{}{
		"size":       vfs.size,
		"sizeClient": vfs.sizeClient,
		"sizeServer": vfs.sizeServer,
	}
}
