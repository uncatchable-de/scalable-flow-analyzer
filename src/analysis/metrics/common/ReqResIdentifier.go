package common

// Analyzes Flows to identify request/response pairs.
// Also responsible for reconstructing flows.

import (
	"analysis/flows"
	"analysis/utils"
	"fmt"
)

// Maximal distance between two ACK responses. If higher, the packets are most likely out of order
const MAXRWIN uint16 = utils.MaxUint16
const MAXWindow uint32 = uint32(MAXRWIN) * 16384 // Max Rwin * Max TCP WINDOW Scale Option (2^14)

type RequestResponse struct {
	Requests     []flows.Packet
	Responses    []flows.Packet
	ClusterIndex int
}

type ReqResIdentifier struct {
	DropUnidirectionalFlows      bool
	ReconstructTCPResponse       bool
	numReconstructedPackets      IntMetric
	statisticReconstructionSpeed *MetricReconstructedPacketsSpeed
	statisticReconstructionSize  *MetricReconstructedPacketsSize
}

func NewReqResIdentifier(dropUnidirectionalFlows, reconstructTCPResponse bool,
	statisticReconstructionSpeed *MetricReconstructedPacketsSpeed,
	statisticReconstructionSize *MetricReconstructedPacketsSize) *ReqResIdentifier {
	var rri = &ReqResIdentifier{
		DropUnidirectionalFlows:      dropUnidirectionalFlows,
		ReconstructTCPResponse:       reconstructTCPResponse,
		numReconstructedPackets:      NewIntMetric(),
		statisticReconstructionSpeed: statisticReconstructionSpeed,
		statisticReconstructionSize:  statisticReconstructionSize,
	}
	return rri
}

func (rri *ReqResIdentifier) reconstructFlow(protocol Protocol, flow *flows.TCPFlow) (numPacketsReconstructed int) {
	type newPacketStruct struct {
		seq   uint32
		size  uint16
		ack   uint32
		index int
	}

	var lastAckNr uint32
	var lastAckTimestamp int64
	var lastAckNrInitialized bool
	var newPackets []newPacketStruct

	// Reconstruct packets based on ACK analysis
	for i, packet := range flow.TCPPacket {
		if !packet.ACK {
			continue
		}
		// Set initial Ack
		if lastAckNrInitialized == false {
			lastAckTimestamp = flow.Packets[i].Timestamp
			lastAckNr = packet.AckNr
			lastAckNrInitialized = true
			continue
		}

		// Calculate transferred bytes (takes also care of overflow)
		var transferredBytes = packet.AckNr - lastAckNr
		// If out of order (ignore packet)
		if transferredBytes > MAXWindow {
			continue
		}
		if transferredBytes > 0 {
			if rri.statisticReconstructionSpeed != nil {
				currentTimestamp := flow.Packets[i].Timestamp
				timediff := int(currentTimestamp-lastAckTimestamp) / 1000 // convert ns to microseconds
				if timediff == 0 {
					timediff = 1
				}
				// Complicated speed calculation to avoid integer errors at division
				speed := (int(transferredBytes) * 1000000) / timediff
				rri.statisticReconstructionSpeed.addReconstructedPacket(protocol, speed, int(transferredBytes))
				rri.statisticReconstructionSize.addReconstructedPacket(protocol, speed, int(transferredBytes))
				lastAckTimestamp = currentTimestamp
			}
			// Split packets in case they are too large for a packet
			for transferredBytes > 0 {
				packetSize := transferredBytes
				if transferredBytes > utils.MaxUint16AsUint32 {
					packetSize = utils.MaxUint16AsUint32
				}

				newPacket := newPacketStruct{
					seq:   packet.AckNr - packetSize,
					ack:   packet.SeqNr,
					size:  uint16(packetSize),
					index: i + len(newPackets),
				}

				newPackets = append(newPackets, newPacket)
				transferredBytes -= packetSize
			}
			lastAckNr = packet.AckNr
		}
	}

	if len(newPackets) == 0 {
		return 0
	}

	// Create new list containing old and new packets
	// Do not insert simply into old list due to inefficient copies during insert
	totalPacketNum := len(flow.Packets) + len(newPackets)
	flowPackets := make([]flows.Packet, totalPacketNum)
	tcpPackets := make([]flows.TCPPacket, totalPacketNum)
	idxNew := 0
	idxOld := 0
	for i := 0; i < totalPacketNum; i++ {
		if len(newPackets) > idxNew && i == newPackets[idxNew].index {
			var timestamp int64
			// Was previously sent (lower timestamp)
			timestamp = flow.Packets[idxOld].Timestamp - 1

			flowPackets[i] = flows.Packet{
				FromClient:    false,
				LengthPayload: newPackets[idxNew].size,
				PacketIdx:     0,
				Timestamp:     timestamp,
			}

			tcpPackets[i] = flows.TCPPacket{
				ACK:   false,
				AckNr: newPackets[idxNew].ack,
				FIN:   false,
				RST:   false,
				SYN:   false,
				SeqNr: newPackets[idxNew].seq,
			}
			idxNew++
		} else {
			flowPackets[i] = flow.Packets[idxOld]
			tcpPackets[i] = flow.TCPPacket[idxOld]
			idxOld++
		}
	}
	flow.Packets = flowPackets
	flow.TCPPacket = tcpPackets

	rri.numReconstructedPackets.AddValue(protocol, len(newPackets))
	return len(newPackets)
}

// isTCPControlPacket returns true if the packet has no payload and is just
// used for TCP handshake, termination or only ACK packet
func isTCPControlPacket(packet flows.Packet, tcpPacket flows.TCPPacket) bool {
	return packet.LengthPayload == 0 && (tcpPacket.ACK || tcpPacket.FIN || tcpPacket.RST || tcpPacket.SYN)
}

// onFlush Identifies the request response pairs per flow. If it is a UDP Flow, tcpPacket is nil.
func (rri *ReqResIdentifier) OnTCPFlush(protocol Protocol, flow *flows.TCPFlow) (reqRes []*RequestResponse, dropFlow bool) {
	var hasRequest bool
	var hasResponse bool

	for _, packet := range flow.Packets {
		if packet.FromClient {
			hasRequest = true
		} else {
			hasResponse = true
		}
		if hasRequest && hasResponse {
			break
		}
	}

	// TCP: reconstruct unidirectional
	if rri.ReconstructTCPResponse && !hasResponse {
		numReconstructed := rri.reconstructFlow(protocol, flow)
		if numReconstructed > 0 {
			hasResponse = true
		}
	}

	// Drop unidirectional: No requests or no response
	if rri.DropUnidirectionalFlows && (!hasRequest || !hasResponse) {
		return reqRes, true
	}

	// Identify Request/Response pairs
	var lastPacketWasRequest = false
	for i, packet := range flow.Packets {
		// Ignore ACK
		if isTCPControlPacket(packet, flow.TCPPacket[i]) {
			continue
		}
		if packet.FromClient {
			// Request
			if !lastPacketWasRequest {
				reqRes = append(reqRes, &RequestResponse{})
			}
			reqRes[len(reqRes)-1].Requests = append(reqRes[len(reqRes)-1].Requests, packet)
			lastPacketWasRequest = true
		} else {
			// Response
			// ignore responses without requests (if we start capturing in the middle of the connection)
			if len(reqRes) == 0 {
				continue
			}
			lastPacketWasRequest = false
			reqRes[len(reqRes)-1].Responses = append(reqRes[len(reqRes)-1].Responses, packet)
		}
	}

	return reqRes, false
}

// onFlush Identifies the request response pairs per flow. If it is a UDP Flow, tcpPacket is nil.
func (rri *ReqResIdentifier) OnUDPFlush(protocol Protocol, flow *flows.UDPFlow) (reqRes []*RequestResponse, dropFlow bool) {
	var hasRequest bool
	var hasResponse bool

	for _, packet := range flow.Packets {
		if packet.FromClient {
			hasRequest = true
		} else {
			hasResponse = true
		}
		if hasRequest && hasResponse {
			break
		}
	}

	// Drop unidirectional: No requests or no response
	if rri.DropUnidirectionalFlows && (!hasRequest || !hasResponse) {
		return reqRes, true
	}

	// Identify Request/Response pairs
	var lastPacketWasRequest = false
	for _, packet := range flow.Packets {
		if packet.FromClient {
			// Request
			if !lastPacketWasRequest {
				reqRes = append(reqRes, &RequestResponse{})
			}
			reqRes[len(reqRes)-1].Requests = append(reqRes[len(reqRes)-1].Requests, packet)
			lastPacketWasRequest = true
		} else {
			// Response
			// ignore responses without requests (if we start capturing in the middle of the connection)
			if len(reqRes) == 0 {
				continue
			}
			lastPacketWasRequest = false
			reqRes[len(reqRes)-1].Responses = append(reqRes[len(reqRes)-1].Responses, packet)
		}
	}

	return reqRes, false
}

func (rri *ReqResIdentifier) PrintStatistic(verbose bool) {
	fmt.Println("Number of reconstructed packets:")
	fmt.Print(rri.numReconstructedPackets.GetStatistics(true))
}
