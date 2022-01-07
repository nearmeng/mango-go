package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
)

// serverStreamTransport ServerStreamTransport interface 的实现，兼容之前的ServerTransport
type serverStreamTransport struct {
	// 兼容原来的serverTransport
	serverTransport
}

// NewServerStreamTransport  新建并返回ServerStreamTransport，里面新建了serverTransport
// 并包含在serverStreamTransport结构体里面进行返回
func NewServerStreamTransport(opt ...ServerTransportOption) ServerStreamTransport {
	// option 默认值
	opts := &ServerTransportOptions{
		RecvUDPPacketBufferSize: 65536,
		SendMsgChannelSize:      100,
		RecvMsgChannelSize:      100,
		IdleTimeout:             time.Minute,
	}

	for _, o := range opt {
		o(opts)
	}
	addrToConn := make(map[string]*tcpconn)
	s := serverTransport{addrToConn: addrToConn, opts: opts, m: &sync.RWMutex{}}
	return &serverStreamTransport{s}
}

func addrToKey(addr net.Addr) string {
	return fmt.Sprintf("%s//%s", addr.Network(), addr.String())
}

// DefaultServerStreamTransport 默认的ServerStreamTransport
var DefaultServerStreamTransport = NewServerStreamTransport()

// ListenAndServe 实现ListenAndServe接口，使用serverTransport里面的ListenAndServe，兼容一应一答和流式
func (st *serverStreamTransport) ListenAndServe(ctx context.Context, opts ...ListenServeOption) error {
	return st.serverTransport.ListenAndServe(ctx, opts...)
}

// Send 提供发送流式消息的接口
func (st *serverStreamTransport) Send(ctx context.Context, req []byte) error {
	msg := codec.Message(ctx)
	addr := msg.RemoteAddr()
	if addr == nil {
		return errs.NewFrameError(errs.RetServerSystemErr, "Remote addr is invalid")
	}
	key := addrToKey(addr)
	st.serverTransport.m.RLock()
	tc, ok := st.serverTransport.addrToConn[key]
	st.serverTransport.m.RUnlock()
	if ok && tc != nil {
		if _, err := tc.rwc.Write(req); err != nil {
			tc.close()
			st.Close(ctx)
			return err
		}
		return nil
	}
	return errs.NewFrameError(errs.RetServerSystemErr, "Can't find conn by addr")
}

// Close ServerStreamTransport 的Close实现，用来清理缓存的链接
func (st *serverStreamTransport) Close(ctx context.Context) {
	msg := codec.Message(ctx)
	addr := msg.RemoteAddr()
	key := addrToKey(addr)
	st.m.Lock()
	delete(st.serverTransport.addrToConn, key)
	st.m.Unlock()
}
