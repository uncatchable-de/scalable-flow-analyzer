package metrics

import (
	"analysis/flows"
)

type Metric interface {
	OnTCPFlush(flow *flows.TCPFlow)
	OnUDPFlush(flow *flows.UDPFlow)
}
