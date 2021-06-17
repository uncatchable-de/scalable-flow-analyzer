package flows

import (
	"analysis/flows"
	"analysis/metrics/common"
)

type MetricRRPs struct{}

func newMetricRRPs() *MetricRRPs {
	return &MetricRRPs{}
}

func (mr *MetricRRPs) calc(flow *flows.Flow, reqRes []*common.RequestResponse) ValueRRPairs {
	var rrps = make([][2]uint16, 0)

	for _, rr := range reqRes {
		requests := rr.Requests
		responses := rr.Responses

		lastCommonIndex := func(a int, b int) int {
			if a < b {
				return a
			} else {
				return b
			}
		}(len(requests), len(responses))

		for i := 0; i < lastCommonIndex; i++ {
			rrps = append(rrps, [2]uint16{requests[i].LengthPayload, responses[i].LengthPayload})
		}
	}

	return ValueRRPairs{
		rrps: rrps,
	}
}

func (mr *MetricRRPs) onFlush(flow *flows.Flow, reqRes []*common.RequestResponse) ExportableValue {
	value := mr.calc(flow, reqRes)
	return value
}

type ValueRRPairs struct {
	// The request response pairs for a flow.
	rrps [][2]uint16
}

func (vr ValueRRPairs) export() map[string]interface{} {
	return map[string]interface{}{
		"rrps": vr.rrps,
	}
}
