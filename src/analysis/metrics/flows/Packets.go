package flows

import (
	"scalable-flow-analyzer/flows"
)

type MetricPackets struct{}

func newMetricPackets() *MetricPackets {
	return &MetricPackets{}
}

func (mp *MetricPackets) calc(flow *flows.Flow) ValuePackets {
	var numPackets, numPacketsClient, numPacketsServer uint32

	packets := flow.Packets
	for i := 0; i < len(packets); i++ {
		p := packets[i]

		numPackets++
		if p.FromClient {
			numPacketsClient++
		} else {
			numPacketsServer++
		}
	}

	return ValuePackets{
		packets:       numPackets,
		packetsServer: numPacketsServer,
		packetsClient: numPacketsClient,
	}
}

func (mp *MetricPackets) onFlush(flow *flows.Flow) ExportableValue {
	value := mp.calc(flow)
	return value
}

type ValuePackets struct {
	// The number of packets that were exchanged.
	packets uint32
	// The number of packets the client sent.
	packetsClient uint32
	// Port number of packets the server sent.
	packetsServer uint32
}

func (vp ValuePackets) export() map[string]interface{} {
	return map[string]interface{}{
		"packets":       vp.packets,
		"packetsClient": vp.packetsClient,
		"packetsServer": vp.packetsServer,
	}
}
