// +build !aix,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd,!solaris,!linux

package connpool

import (
	"errors"
	"net"
	"time"
)

func checkConnErr(conn net.Conn, buf []byte) error {
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	n, err := conn.Read(buf)
	// 空闲的连接不应该读出数据，黏包处理错误
	if err == nil || n > 0 {
		return errors.New("unexpected read from socket")
	}
	// 空闲连接正常状态返回timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		conn.SetReadDeadline(time.Time{})
		return nil
	}
	// 其它连接错误，包括连接已关闭
	return err
}

func checkConnErrUnblock(conn net.Conn, buf []byte) error {
	// 暂不支持非阻塞模式
	return nil
}
