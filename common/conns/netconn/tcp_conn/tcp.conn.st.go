package tcp_conn

import (
	"encoding/binary"
	"io"
	"net"

	"server.slg.com/common/conns/netconn"
	"server.slg.com/common/conns/netconn/packets"
)

// NetConn TCP 网络连接封装，实现 NetConnI 接口，提供基于 TCP 的数据包读写能力
type NetConn struct {
	conn net.Conn
}

var _ netconn.NetConnI = (*NetConn)(nil)

func NewNetConn(c net.Conn) *NetConn {
	return &NetConn{conn: c}
}

func (n NetConn) Close() error {
	return n.conn.Close()
}

func (n NetConn) ReadFromConn() (*packets.Packet, error) {
	headerBuf := packets.GetHeadBuf(packets.TcpHeaderSize)
	defer packets.PutHeadBuf(headerBuf) // 读取完归还
	if _, err := io.ReadFull(n.conn, headerBuf); err != nil {
		return nil, err
	}

	// 解析长度和协议ID
	length := binary.BigEndian.Uint32(headerBuf[0:packets.TcpLengthSizeTail])
	seq := binary.BigEndian.Uint32(headerBuf[packets.TcpLengthSizeTail:packets.TcpSeqSizeTail])
	msgID := binary.BigEndian.Uint32(headerBuf[packets.TcpSeqSizeTail:packets.TcpMsgIdSizeTail])

	// 读取body
	bodyBuf := packets.GetMsgBuf(int(length - packets.TcpHeaderSize))

	if _, err := io.ReadFull(n.conn, bodyBuf); err != nil {
		packets.PutMsgBuf(bodyBuf)
		return nil, err
	}

	return &packets.Packet{
		Length: length,
		Seq:    seq,
		MsgID:  msgID,
		Body:   bodyBuf,
	}, nil
}

func (n NetConn) WriteToConn(seq uint32, packetData *packets.Packet) error {
	totalSize := packets.TcpHeaderSize + len(packetData.Body)

	// 从池里获取缓冲区
	buf := packets.GetMsgBuf(totalSize)
	defer packets.PutMsgBuf(buf) // 发送完归还

	binary.BigEndian.PutUint32(buf[:packets.TcpLengthSizeTail], uint32(totalSize))
	binary.BigEndian.PutUint32(buf[packets.TcpLengthSizeTail:packets.TcpSeqSizeTail], seq)
	binary.BigEndian.PutUint32(buf[packets.TcpSeqSizeTail:packets.TcpMsgIdSizeTail], packetData.MsgID)
	copy(buf[packets.TcpMsgIdSizeTail:], packetData.Body)

	_, err := n.conn.Write(buf)
	return err
}

func (n NetConn) RemoteAddr() string {
	return n.conn.RemoteAddr().String()
}
