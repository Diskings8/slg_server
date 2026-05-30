package netconn

import "server.slg.com/common/conns/netconn/packets"

type NetConnI interface {
	Close() error
	ReadFromConn() (*packets.Packet, error)
	WriteToConn(seq uint32, packetData *packets.Packet) error
	RemoteAddr() string
}
