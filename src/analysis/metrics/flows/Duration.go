package flows

import (
	"scalable-flow-analyzer/flows"
)

type MetricFlowDuration struct{}

func newMetricFlowDuration() *MetricFlowDuration {
	return &MetricFlowDuration{}
}

func (mfd *MetricFlowDuration) calc(flow *flows.Flow) ValueFlowDuration {
	packets := flow.Packets

	start := packets[0].Timestamp
	end := packets[len(packets)-1].Timestamp

	return ValueFlowDuration{
		start:    start,
		end:      end,
		duration: end - start,
	}
}

func (mfd *MetricFlowDuration) onFlush(flow *flows.Flow) ExportableValue {
	value := mfd.calc(flow)
	return value
}

type ValueFlowDuration struct {
	// Start time as unix timestamp.
	start int64
	// End time as unix timestamp.
	end int64
	// Duration in nano seconds.
	duration int64
}

func (vfd ValueFlowDuration) export() map[string]interface{} {
	return map[string]interface{}{
		"start":    vfd.start,
		"end":      vfd.end,
		"duration": vfd.duration,
	}
}
