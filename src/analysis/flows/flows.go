package flows

// TCPTimeout in Nanoseconds
var TCPTimeout int64

// TCPRstTimeout in Nanoseconds
var TCPRstTimeout int64

// TCPFinTimeout in Nanoseconds
var TCPFinTimeout int64

// UDPTimeout in Nanoseconds
var UDPTimeout int64

// TCP Protocol
const TCP uint8 = 1

// UDP Protocol
const UDP uint8 = 0

func GetProtocolString(protocol uint8) string {
	switch protocol {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	default:
		return "Unknown"
	}
}

// FlowKeyType defines the type of the key to identify a flow
// based on its protocol and sender and receiver ips and ports.
// Used for flow construction (basically a hash: uint64)
type FlowKeyType uint64

type PacketInformation struct {
	PacketIdx     int64
	FlowKey       FlowKeyType
	SrcPort       uint16
	DstPort       uint16
	PayloadLength uint16
	TCPAckNr      uint32
	TCPSeqNr      uint32
	SrcIP         uint64
	DstIP         uint64
	Timestamp     int64
	TCPFIN        bool
	TCPACK        bool
	TCPRST        bool
	TCPSYN        bool
	HasTCP        bool
	HasUDP        bool
}

// Packet defines a TCP or UDP Packet
// Take care of field order to ensure no wasted memory due to memalign
type Packet struct {
	Timestamp     int64
	PacketIdx     int64
	LengthPayload uint16
	FromClient    bool
}

type TCPPacket struct {
	SeqNr uint32
	AckNr uint32
	ACK   bool
	SYN   bool
	RST   bool
	FIN   bool
}

// Flow is a connection between two application instances.
type Flow struct {
	// The client is the one who initiates the connection or based on lower port number
	// In case no SYN packets are processed, client is the one who sends the first packet
	FlowKey      FlowKeyType
	Timeout      int64
	ClusterIndex int
	ClientAddr   uint64
	ServerAddr   uint64
	ClientPort   uint16
	ServerPort   uint16
	Protocol     uint8 // Indicates transport protocol (TCP/UDP)
	Packets      []Packet
}

// TCPFlow is a Flow with special fields for TCP connections
type TCPFlow struct {
	Flow
	TCPPacket     []TCPPacket
	RSTIndex      int32
	FirstFINIndex int32
}

// UDPFlow is a Flow with special fields for UDP connections
type UDPFlow struct {
	Flow
}

// NewTCPFlow creates a new TCP Flow with default values
func NewTCPFlow(packetInfo PacketInformation) *TCPFlow {
	f := TCPFlow{
		Flow: Flow{
			Protocol: TCP,
			FlowKey:  packetInfo.FlowKey,
		},
		FirstFINIndex: -1,
		RSTIndex:      -1,
	}

	f.setClientServer(packetInfo)
	f.AddPacket(packetInfo)
	// if not a syn packet then set Client and server based on first package
	return &f
}

// NewUDPFlow creates a new TCP Flow with default values
func NewUDPFlow(packetInfo PacketInformation) *UDPFlow {
	f := UDPFlow{
		Flow: Flow{
			Protocol: UDP,
			FlowKey:  packetInfo.FlowKey,
		},
	}
	f.setClientServer(packetInfo)
	f.AddPacket(packetInfo)
	return &f
}

func (f *Flow) addPacket(packetInfo PacketInformation) {
	var newPacket = Packet{
		FromClient:    f.ClientAddr == packetInfo.SrcIP && f.ClientPort == packetInfo.SrcPort,
		PacketIdx:     packetInfo.PacketIdx,
		Timestamp:     packetInfo.Timestamp,
		LengthPayload: packetInfo.PayloadLength}
	f.Packets = append(f.Packets, newPacket)
}

// AddPacket to TCP Flow
func (f *TCPFlow) AddPacket(packetInfo PacketInformation) {
	f.Flow.addPacket(packetInfo) // super method
	f.TCPPacket = append(f.TCPPacket, TCPPacket{
		SeqNr: packetInfo.TCPSeqNr,
		AckNr: packetInfo.TCPAckNr,
		ACK:   packetInfo.TCPACK,
		FIN:   packetInfo.TCPFIN,
		RST:   packetInfo.TCPRST,
		SYN:   packetInfo.TCPSYN})
	switch {
	case packetInfo.TCPRST:
		f.RSTIndex = int32(len(f.Packets) - 1)
		f.Timeout = packetInfo.Timestamp + TCPRstTimeout
	case packetInfo.TCPFIN && f.FirstFINIndex == -1:
		f.FirstFINIndex = int32(len(f.Packets) - 1)
		f.Timeout = packetInfo.Timestamp + TCPFinTimeout
	default:
		f.Timeout = packetInfo.Timestamp + TCPTimeout
	}
}

func (f *TCPFlow) setClientServer(packetInfo PacketInformation) {
	switch {
	case packetInfo.TCPSYN && !packetInfo.TCPACK:
		// From Client
		f.ClientAddr = packetInfo.SrcIP
		f.ClientPort = packetInfo.SrcPort
		f.ServerAddr = packetInfo.DstIP
		f.ServerPort = packetInfo.DstPort
	case packetInfo.TCPSYN && packetInfo.TCPACK:
		// From Server
		f.ClientAddr = packetInfo.DstIP
		f.ClientPort = packetInfo.DstPort
		f.ServerAddr = packetInfo.SrcIP
		f.ServerPort = packetInfo.SrcPort
	case packetInfo.SrcPort <= 49151 && packetInfo.SrcPort < packetInfo.DstPort:
		// From Server
		f.ClientAddr = packetInfo.DstIP
		f.ClientPort = packetInfo.DstPort
		f.ServerAddr = packetInfo.SrcIP
		f.ServerPort = packetInfo.SrcPort
	default:
		// From Client
		f.ClientAddr = packetInfo.SrcIP
		f.ClientPort = packetInfo.SrcPort
		f.ServerAddr = packetInfo.DstIP
		f.ServerPort = packetInfo.DstPort
	}
}

// AddPacket to UDP Flow
func (f *UDPFlow) AddPacket(packetInfo PacketInformation) {
	f.Flow.addPacket(packetInfo) // super method
	f.Timeout = packetInfo.Timestamp + UDPTimeout
}

func (f *UDPFlow) setClientServer(packetInfo PacketInformation) {
	if packetInfo.SrcPort <= 49151 && packetInfo.SrcPort < packetInfo.DstPort {
		// From Server
		f.ClientAddr = packetInfo.DstIP
		f.ClientPort = packetInfo.DstPort
		f.ServerAddr = packetInfo.SrcIP
		f.ServerPort = packetInfo.SrcPort
	} else {
		// From Client
		f.ClientAddr = packetInfo.SrcIP
		f.ClientPort = packetInfo.SrcPort
		f.ServerAddr = packetInfo.DstIP
		f.ServerPort = packetInfo.DstPort
	}
}
