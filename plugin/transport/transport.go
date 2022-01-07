package transport

import "net"

type Conn interface {
	GetConnID() uint64
	GetLocalAddr() (addr net.Addr)
	GetRemoteAddr() (addr net.Addr)
	Send(data []byte) error
	Read(targetBuff []byte) (int, error)
	Close(active bool) error
}

type EventHandler interface {
	OnConnOpened(conn Conn)
	OnConnClosed(conn Conn, active bool)
	OnData(conn Conn, data []byte)
}

type Options struct {
	EventHandler EventHandler
}

type Transport interface {
	Init(o Options) error
	Uninit() error
}
