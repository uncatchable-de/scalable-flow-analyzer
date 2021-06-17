package common

// Protocol identifies a network protocol based on its Transport Protocol and Server Port
type Protocol struct {
	Protocol    uint8
	Port        uint16
	ProtocolKey ProtocolKeyType
}
