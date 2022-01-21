package app

import (
	"github.com/nearmeng/mango-go/plugin/transport"
	"github.com/nearmeng/mango-go/server_base/msg"
)

type eventTcp struct {
}

func (*eventTcp) OnConnOpened(conn transport.Conn) {
	msg.OnClientConnOpened(conn)
}

func (*eventTcp) OnConnClosed(conn transport.Conn, active bool) {
	msg.OnClientConnClosed(conn, active)
}

func (*eventTcp) OnData(conn transport.Conn, data []byte) {
	msg.RecvClientMsg(conn, data)
}
