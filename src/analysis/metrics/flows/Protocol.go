package flows

import (
	"scalable-flow-analyzer/flows"
)

type MetricProtocol struct{}

func newMetricProtocol() *MetricProtocol {
	return &MetricProtocol{}
}

func (mp *MetricProtocol) onFlush(flow *flows.Flow) ExportableValue {
	value := ValueProtocol{
		protocol:      flows.GetProtocolString(flow.Protocol),
		portClient:    flow.ClientPort,
		portServer:    flow.ServerPort,
		addressClient: int64(flow.ClientAddr),
		addressServer: int64(flow.ServerAddr),
	}

	return value
}

type ValueProtocol struct {
	// The name of the layer 4 protocol used.
	protocol string
	// Port number the client used.
	portClient uint16
	// Port number the server used.
	portServer uint16
	// Address the client used. Conversion to int64 needed for elasticsearch.
	addressClient int64
	// Address the server used. Conversion to int64 needed for elasticsearch.
	addressServer int64
}

func (vp ValueProtocol) export() map[string]interface{} {
	return map[string]interface{}{
		"protocol":      vp.protocol,
		"portClient":    vp.portClient,
		"portServer":    vp.portServer,
		"addressClient": vp.addressClient,
		"addressServer": vp.addressServer,
	}
}
