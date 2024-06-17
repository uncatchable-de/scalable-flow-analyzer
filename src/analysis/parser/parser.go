package parser

import (
	"scalable-flow-analyzer/flows"
	"scalable-flow-analyzer/pool"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/cespare/xxhash"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// parserChannelSize defines the Size of the channel to the Parser
const parserChannelSize = 40000

// packetDataCacheSize is the batching size of the packets sent to the Parsers
const packetDataCacheSize = 1600

const ringBufferFlushChannelSize = 200

// Parser multithreads parsing of packets
type Parser struct {
	numFlowThreads       uint64
	parsePacketDataCache packetDataCache
	pool                 *pool.Pools
	samplingrate         float64
	numParserChannel     int
	parserChannel        []chan [packetDataCacheSize]PacketData

	ringbufferUsedlist     []bool // Same size as ringbuffer. Indicates whether a ringbuffer entry is used or not
	ringbuffer             []flows.PacketInformation
	ringbufferStart        int64
	ringbufferSize         int64
	ringbufferFlushChannel chan bool

	wgParserThreads   sync.WaitGroup // Waitgroup to wait until parser are finished
	wgRingbufferFlush sync.WaitGroup // Waitgroup to wait until Ringbuffer is flushed
}

// PacketData contains the basic information from the packet source
type PacketData struct {
	Data      []byte
	Timestamp int64
	PacketIdx int64
}

type packetDataCache struct {
	buf [packetDataCacheSize]PacketData
	pos int
}

// NewParser returns a new parser
func NewParser(p *pool.Pools, sortingRingBufferSize int64, numParserThreads int, samplingrate float64, numParserChannel int) *Parser {
	var parser = &Parser{
		pool:                   p,
		samplingrate:           samplingrate,
		numParserChannel:       int(math.Min(float64(numParserChannel), float64(numParserThreads))),
		parsePacketDataCache:   packetDataCache{},
		ringbufferUsedlist:     make([]bool, sortingRingBufferSize),
		ringbuffer:             make([]flows.PacketInformation, sortingRingBufferSize),
		ringbufferStart:        1,
		ringbufferSize:         sortingRingBufferSize,
		ringbufferFlushChannel: make(chan bool, ringBufferFlushChannelSize),
		numFlowThreads:         uint64(p.GetNumFlowThreads()),
	}
	parser.wgParserThreads.Add(numParserThreads)
	parser.parserChannel = make([]chan [packetDataCacheSize]PacketData, parser.numParserChannel)
	for i := 0; i < numParserChannel; i++ {
		parser.parserChannel[i] = make(chan [packetDataCacheSize]PacketData, parserChannelSize)
	}
	for i := 0; i < numParserThreads; i++ {
		go parser.parsePacket(parser.parserChannel[i%parser.numParserChannel], i)
	}
	parser.wgRingbufferFlush.Add(1)
	go parser.flushRingbuffer()
	return parser
}

// Close Parser and flush out all packets to the pool
func (p *Parser) Close() {
	// Flush to parser
	tmpPacketsCache := packetDataCache{}
	copy(tmpPacketsCache.buf[:p.parsePacketDataCache.pos], p.parsePacketDataCache.buf[:p.parsePacketDataCache.pos])
	p.parserChannel[0] <- tmpPacketsCache.buf
	// Close Parser
	for i := 0; i < p.numParserChannel; i++ {
		close(p.parserChannel[i])
	}
	p.wgParserThreads.Wait()

	// Ensure to flush out all remaining packets from the sorting ringbuffer
	p.ringbufferFlushChannel <- true
	close(p.ringbufferFlushChannel)
	p.wgRingbufferFlush.Wait()
}

// ParsePacket adds a packet to the parser (buffered)
func (p *Parser) ParsePacket(data []byte, packetIdx, packetTimestamp int64) {
	p.parsePacketDataCache.buf[p.parsePacketDataCache.pos] = PacketData{Data: data, PacketIdx: packetIdx, Timestamp: packetTimestamp}
	p.parsePacketDataCache.pos++
	if p.parsePacketDataCache.pos == packetDataCacheSize {
		p.parserChannel[rand.Intn(p.numParserChannel)] <- p.parsePacketDataCache.buf
		p.parsePacketDataCache.pos = 0
	}
}

// parsePacket is the internal method, called when the internal cache/buffer is full
func (p *Parser) parsePacket(channel chan [packetDataCacheSize]PacketData, parserIndex int) {
	var dot1q layers.Dot1Q
	var gre layers.GRE
	var eth layers.Ethernet

	var ipv4 layers.IPv4
	var ipv6 layers.IPv6
	var ipv6e layers.IPv6ExtensionSkipper
	var tcp layers.TCP
	var udp layers.UDP
	var samplingModulo uint64 = 1
	// ensure that modulo is really 1, when 100 percent sampling rate (due to float conversion)
	if p.samplingrate != 100 {
		samplingModulo = uint64(float64(p.numFlowThreads) * (100 / p.samplingrate))
	}

	parser := gopacket.NewDecodingLayerParser(
		layers.LayerTypeEthernet,
		&dot1q, &eth, &gre, &ipv4, &ipv6, &ipv6e, &tcp, &udp)
	parserIPv4 := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv4, &ipv4, &tcp, &udp)
	parserIPv6 := gopacket.NewDecodingLayerParser(layers.LayerTypeIPv6, &ipv6, &ipv6e, &tcp, &udp)
	var decoded []gopacket.LayerType
	for packets := range channel {
		for _, packet := range &packets {
			// Ignore empty packets from last flush
			if packet.PacketIdx == 0 {
				continue
			}
			_ = parserIPv4.DecodeLayers(packet.Data, &decoded)
			if len(decoded) < 2 {
				_ = parser.DecodeLayers(packet.Data, &decoded)
				if len(decoded) < 2 {
					_ = parserIPv6.DecodeLayers(packet.Data, &decoded)
				}
			}
			packetInfo := flows.PacketInformation{Timestamp: packet.Timestamp, PacketIdx: packet.PacketIdx}
			var ipLength uint16
			for _, layerType := range decoded {
				switch layerType {
				case layers.LayerTypeIPv4:
					ipLength = ipv4.Length - (uint16(ipv4.IHL) * 4)
					packetInfo.SrcIP = xxhash.Sum64(ipv4.SrcIP)
					packetInfo.DstIP = xxhash.Sum64(ipv4.DstIP)
				case layers.LayerTypeIPv6:
					ipLength = ipv6.Length
					// if zero
					if ipLength == 0 {
						fmt.Println("Jumbogram detected. Currently unsupported.")
					}
					// Subtract possible extension header length
					if len(ipv6e.Contents) != 0 {
						// since ipv6 can contain more than one extension header: search for last extens
						// TODO: search for last extension and remove them from iplength
						ipLength -= uint16(len(ipv6e.Contents))
						ipv6e.Contents = make([]byte, 0)
					}
					packetInfo.SrcIP = xxhash.Sum64(ipv6.SrcIP)
					packetInfo.DstIP = xxhash.Sum64(ipv6.DstIP)
				case layers.LayerTypeTCP:
					packetInfo.HasTCP = true
					packetInfo.TCPSYN = tcp.SYN
					packetInfo.TCPACK = tcp.ACK
					packetInfo.TCPRST = tcp.RST
					packetInfo.TCPFIN = tcp.FIN
					packetInfo.SrcPort = uint16(tcp.SrcPort)
					packetInfo.DstPort = uint16(tcp.DstPort)
					packetInfo.TCPSeqNr = tcp.Seq
					packetInfo.TCPAckNr = tcp.Ack
					packetInfo.PayloadLength = ipLength - (uint16(tcp.DataOffset) * 4) // Data offset in 32 bits words
					packetInfo.FlowKey = GetFlowKey(packetInfo.SrcIP, packetInfo.DstIP, flows.TCP, packetInfo.SrcPort, packetInfo.DstPort)
				case layers.LayerTypeUDP:
					packetInfo.HasUDP = true
					packetInfo.SrcPort = uint16(udp.SrcPort)
					packetInfo.DstPort = uint16(udp.DstPort)
					packetInfo.PayloadLength = udp.Length
					packetInfo.FlowKey = GetFlowKey(packetInfo.SrcIP, packetInfo.DstIP, flows.UDP, packetInfo.SrcPort, packetInfo.DstPort)
				}
			}

			for packetInfo.PacketIdx-p.ringbufferStart > p.ringbufferSize {
				time.Sleep(1 * time.Second)
				fmt.Println("Parser", parserIndex, ": Sleep for 1s due to missing space in ringbuffer.")
				fmt.Println("Parser", parserIndex, ": Please increase sortingRingBufferSize variable or increase number of pool to speed up flushing if this happens more often.")
			}
			// Sampling
			if uint64(packetInfo.FlowKey)%samplingModulo > p.numFlowThreads {
				packetInfo.HasTCP = false
				packetInfo.HasUDP = false
			}
			ringBufferIndex := packetInfo.PacketIdx % p.ringbufferSize
			p.ringbuffer[ringBufferIndex] = packetInfo
			p.ringbufferUsedlist[ringBufferIndex] = true
		}
		if rand.Intn(100) <= 5 {
			p.ringbufferFlushChannel <- true
		}
	}
	p.wgParserThreads.Done()
}

// flushRingbuffer checks if packets can be flushed out to the processing unit.
func (p *Parser) flushRingbuffer() {
	for range p.ringbufferFlushChannel {
		// Go through ringbuffer and flush out all available packets
		for i := p.ringbufferStart; true; i++ {
			ringBufferIndex := i % p.ringbufferSize
			if !p.ringbufferUsedlist[ringBufferIndex] {
				p.ringbufferStart = i
				break
			}
			if p.ringbuffer[ringBufferIndex].HasTCP {
				p.pool.AddTCPPacket(&p.ringbuffer[ringBufferIndex])
			} else if p.ringbuffer[ringBufferIndex].HasUDP {
				p.pool.AddUDPPacket(&p.ringbuffer[ringBufferIndex])
			}
			p.ringbufferUsedlist[ringBufferIndex] = false
		}
	}
	p.wgRingbufferFlush.Done()
}

// GetFlowKey returns the Flow key. Is symmetric so A:46254<-->B:80 returns the same key in both directions
func GetFlowKey(srcIP, dstIP uint64, protocol uint8, srcPort, dstPort uint16) flows.FlowKeyType {
	var app = make([]byte, 10)
	binary.LittleEndian.PutUint16(app, srcPort)
	binary.LittleEndian.PutUint64(app[2:], srcIP)
	var hashSrc = xxhash.Sum64(app)

	binary.LittleEndian.PutUint16(app, dstPort)
	binary.LittleEndian.PutUint64(app[2:], dstIP)
	var hashDst = xxhash.Sum64(app)
	return flows.FlowKeyType(hashSrc + uint64(protocol) + hashDst)
}
