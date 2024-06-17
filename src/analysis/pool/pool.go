package pool

// This file contains everything needed to manage a pool of flows.
// This file is internal only, other components use pools.

import (
	"scalable-flow-analyzer/flows"
	"scalable-flow-analyzer/metrics"
	"sync"
)

// Pool is a collection of Flows previously seen
type pool struct {
	addTCPPacketCache   packetInformationCache
	addTCPPacketChannel chan [packetInformationCacheSize]flows.PacketInformation
	addUDPPacketCache   packetInformationCache
	addUDPPacketChannel chan [packetInformationCacheSize]flows.PacketInformation
	tcpFlows            map[flows.FlowKeyType]*flows.TCPFlow // each flowthread has its own map to avoid concurrency
	udpFlows            map[flows.FlowKeyType]*flows.UDPFlow // each flowthread has its own map to avoid concurrency
	metrics             []metrics.Metric
	currentTCPTime      int64
	currentUDPTime      int64
	wgAddPacket         sync.WaitGroup
	tcpFlowsLock        sync.Mutex // Lock synchronizes with flushing
	udpFlowsLock        sync.Mutex // Lock synchronizes with flushing
	tcpFilter           [65536]bool
	udpFilter           [65536]bool
	tcpDropIncomplete   bool
}

type packetInformationCache struct {
	buf [packetInformationCacheSize]flows.PacketInformation
	pos int
}

// NewPool creates an empty pool of flows
func newPool(tcpFilter, udpFilter *[65536]bool, tcpDropIncomplete bool) *pool {
	p := pool{tcpFilter: *tcpFilter, udpFilter: *udpFilter, tcpDropIncomplete: tcpDropIncomplete}

	// Start goroutines to add packets
	p.wgAddPacket.Add(1)
	p.tcpFlows = make(map[flows.FlowKeyType]*flows.TCPFlow)
	p.addTCPPacketChannel = make(chan [packetInformationCacheSize]flows.PacketInformation, addPacketChannelSize)
	go p.addTCPPackets()

	p.wgAddPacket.Add(1)
	p.udpFlows = make(map[flows.FlowKeyType]*flows.UDPFlow)
	p.addUDPPacketChannel = make(chan [packetInformationCacheSize]flows.PacketInformation, addPacketChannelSize)
	go p.addUDPPackets()

	return &p
}

// ClosePool adds all remaining packets to pool and then flushes all packets to the metrics.
func (p *pool) close() {
	// Write remaining packets from channels to flows
	tmp := [packetInformationCacheSize]flows.PacketInformation{}
	copy(tmp[:p.addTCPPacketCache.pos], p.addTCPPacketCache.buf[:p.addTCPPacketCache.pos])
	p.addTCPPacketChannel <- tmp
	close(p.addTCPPacketChannel)
	tmp = [packetInformationCacheSize]flows.PacketInformation{}
	copy(tmp[:p.addUDPPacketCache.pos], p.addUDPPacketCache.buf[:p.addUDPPacketCache.pos])
	p.addUDPPacketChannel <- tmp
	close(p.addUDPPacketChannel)

	p.wgAddPacket.Wait()
}

func (p *pool) addTCPPacket(packet *flows.PacketInformation) {
	p.addTCPPacketCache.buf[p.addTCPPacketCache.pos] = *packet
	p.addTCPPacketCache.pos++
	if p.addTCPPacketCache.pos == packetInformationCacheSize {
		p.addTCPPacketChannel <- p.addTCPPacketCache.buf
		p.addTCPPacketCache.pos = 0
	}
}

func (p *pool) addTCPPackets() {
	for tcpPackets := range p.addTCPPacketChannel {
		p.tcpFlowsLock.Lock()
		for _, tcpPacket := range &tcpPackets {
			if tcpPacket.PacketIdx == 0 {
				continue
			}
			if !p.tcpFilter[tcpPacket.SrcPort] && !p.tcpFilter[tcpPacket.DstPort] {
				continue
			}
			p.currentTCPTime = tcpPacket.Timestamp
			flow, flowExists := p.tcpFlows[tcpPacket.FlowKey]
			// Check if connection is timedout or a new connection is establishing
			if flowExists {
				// Check if connection timed out. Exception: TCP RST is set, then it belongs to current flow (e.g. tearing down due to timeout)
				if !tcpPacket.TCPRST && p.flushTCPFlow(flow, false) {
					flowExists = false
				}

				// If new TCP Connection and Old Flow was terminated: Force flush
				if flowExists && tcpPacket.TCPSYN && (flow.FirstFINIndex != -1 || flow.RSTIndex != -1) {
					p.flushTCPFlow(flow, true)
					flowExists = false
				}
			}
			// Create new flow
			if !flowExists {
				flow = flows.NewTCPFlow(tcpPacket)
				p.tcpFlows[flow.FlowKey] = flow
			} else {
				// Add packet to existing flow
				flow.AddPacket(tcpPacket)
			}
		}
		p.tcpFlowsLock.Unlock()
	}
	p.wgAddPacket.Done()
}

func (p *pool) addUDPPacket(packet *flows.PacketInformation) {
	p.addUDPPacketCache.buf[p.addUDPPacketCache.pos] = *packet
	p.addUDPPacketCache.pos++
	if p.addUDPPacketCache.pos == packetInformationCacheSize {
		p.addUDPPacketChannel <- p.addUDPPacketCache.buf
		p.addUDPPacketCache.pos = 0
	}
}

func (p *pool) addUDPPackets() {
	for udpPackets := range p.addUDPPacketChannel {
		p.udpFlowsLock.Lock()
		for _, udpPacket := range &udpPackets {
			if udpPacket.PacketIdx == 0 {
				continue
			}
			if !p.udpFilter[udpPacket.SrcPort] && !p.udpFilter[udpPacket.DstPort] {
				continue
			}
			p.currentUDPTime = udpPacket.Timestamp
			flow, flowExists := p.udpFlows[udpPacket.FlowKey]
			// Check if connection is timedout
			if flowExists && p.flushUDPFlow(flow, false) {
				flowExists = false
			}

			// Create new flow
			if !flowExists {
				flow = flows.NewUDPFlow(udpPacket)
				p.udpFlows[flow.FlowKey] = flow
			} else {
				// Add packet to existing flow
				flow.AddPacket(udpPacket)
			}
		}
		p.udpFlowsLock.Unlock()
	}
	p.wgAddPacket.Done()
}

// flushTCPFlow flushes a TCP connection if has timed out, or force=true. Returns whether connection can be removed.
func (p *pool) flushTCPFlow(flow *flows.TCPFlow, force bool) bool {
	// Needs Flush
	if force || p.currentTCPTime > flow.Flow.Timeout {
		// Ignore filtered ports
		// Ignore incomplete flows (only SYN must be set)
		if !p.tcpFilter[flow.ServerPort] || (p.tcpDropIncomplete && (!flow.TCPPacket[0].SYN || flow.TCPPacket[0].ACK)) {
			return true
		}

		for _, metric := range p.metrics {
			metric.OnTCPFlush(flow)
		}

		return true
	}
	return false
}

// flushUDPFlow flushes a UDP connection if has timed out, or force=true. Returns whether connection has been flushed.
func (p *pool) flushUDPFlow(flow *flows.UDPFlow, force bool) bool {
	// Needs Flush
	if force || p.currentUDPTime > flow.Flow.Timeout {
		// Ignore filtered ports
		if !p.udpFilter[flow.ServerPort] {
			return true
		}

		for _, metric := range p.metrics {
			metric.OnUDPFlush(flow)
		}

		return true
	}
	return false
}

// Flush will flush all closed connections
func (p *pool) flush(force bool, wgFlush *sync.WaitGroup, tcpFlushed, tcpCount, udpFlushed, udpCount *int64, counterLock *sync.Mutex) {
	// Start concurrent threads which can check if Flows needs flushing concurrently
	wgFlush.Add(1)
	go func(force bool, wgFlush *sync.WaitGroup) {
		p.tcpFlowsLock.Lock()
		counterLock.Lock()
		*tcpCount += int64(len(p.tcpFlows))
		counterLock.Unlock()
		var flushed int64
		for _, flow := range p.tcpFlows {
			if p.flushTCPFlow(flow, force) {
				delete(p.tcpFlows, flow.FlowKey)
				flushed++
			}
		}
		p.tcpFlowsLock.Unlock()
		counterLock.Lock()
		*tcpFlushed += flushed
		counterLock.Unlock()
		wgFlush.Done()
	}(force, wgFlush)

	wgFlush.Add(1)
	go func(force bool, wgFlush *sync.WaitGroup) {
		p.udpFlowsLock.Lock()
		counterLock.Lock()
		*udpCount += int64(len(p.udpFlows))
		counterLock.Unlock()
		var flushed int64
		for _, flow := range p.udpFlows {
			if p.flushUDPFlow(flow, force) {
				delete(p.udpFlows, flow.FlowKey)
				flushed++
			}
		}
		p.udpFlowsLock.Unlock()
		counterLock.Lock()
		*udpFlushed += flushed
		counterLock.Unlock()
		wgFlush.Done()
	}(force, wgFlush)
}

// registerMetric registers a Metric which shall be called on flush
func (p *pool) registerMetric(metric metrics.Metric) {
	p.metrics = append(p.metrics, metric)
}

// printStatistics print so>me statistics about the pool
func (p *pool) printStatistics(numTCPFlows, numTCPPackets, numUDPFlows, numUDPPackets *int64, counterLock *sync.Mutex) {
	var numFlows int64
	var numPackets int64
	p.tcpFlowsLock.Lock()
	numFlows += int64(len(p.tcpFlows))
	for _, flow := range p.tcpFlows {
		numPackets += int64(len(flow.Packets))
	}
	p.tcpFlowsLock.Unlock()
	counterLock.Lock()
	*numTCPFlows += numFlows
	*numTCPPackets += numPackets
	counterLock.Unlock()

	numFlows = 0
	numPackets = 0
	p.udpFlowsLock.Lock()
	numFlows += int64(len(p.udpFlows))
	for _, flow := range p.udpFlows {
		numPackets += int64(len(flow.Packets))
	}
	p.udpFlowsLock.Unlock()
	counterLock.Lock()
	*numUDPFlows += numFlows
	*numUDPPackets += numPackets
	counterLock.Unlock()
}
