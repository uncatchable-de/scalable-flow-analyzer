package metrics

import (
	"scalable-flow-analyzer/flows"
)

type Metric interface {
	OnTCPFlush(flow *flows.TCPFlow)
	OnUDPFlush(flow *flows.UDPFlow)
}
