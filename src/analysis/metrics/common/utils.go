package common

import (
	"scalable-flow-analyzer/flows"
	"encoding/binary"
	"log"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
)

// ProtocolKeyType is the hashed interpretation of an application protocol (TCP/UDP + Port)
type ProtocolKeyType uint64

func GetProtocolKey(protocolString string) ProtocolKeyType {
	splits := strings.SplitN(protocolString, "_", 2)

	var protocol uint8
	switch strings.ToLower(splits[0]) {
	case "tcp":
		protocol = flows.TCP
	case "udp":
		protocol = flows.UDP
	}

	port, err := strconv.ParseUint(splits[1], 10, 16)
	if err != nil {
		log.Fatalln("protocolString is not well formatted", protocolString)
	}
	var serverPort = uint16(port)

	var bytesBuffer = make([]byte, 3)
	binary.LittleEndian.PutUint16(bytesBuffer[0:2], serverPort)
	bytesBuffer[2] = protocol
	return ProtocolKeyType(xxhash.Sum64(bytesBuffer))
}

func GetProtocol(flow *flows.Flow) Protocol {
	var bytesBuffer = make([]byte, 3)
	binary.LittleEndian.PutUint16(bytesBuffer[0:2], flow.ServerPort)
	bytesBuffer[2] = flow.Protocol
	return Protocol{Protocol: flow.Protocol, Port: flow.ServerPort, ProtocolKey: ProtocolKeyType(xxhash.Sum64(bytesBuffer))}
}

func (protocol Protocol) GetProtocolString() string {
	return flows.GetProtocolString(protocol.Protocol) + "_" + strconv.Itoa(int(protocol.Port))
}
