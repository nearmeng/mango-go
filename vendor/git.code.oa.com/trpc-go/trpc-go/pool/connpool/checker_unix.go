// +build aix darwin dragonfly freebsd netbsd openbsd solaris linux

package connpool

import (
	"errors"
	"io"
	"net"
	"syscall"

	"git.code.oa.com/trpc-go/trpc-go/internal/report"
)

func checkConnErr(conn net.Conn, buf []byte) error {
	return checkConnErrUnblock(conn, buf)
}

func checkConnErrUnblock(conn net.Conn, buf []byte) error {
	sysConn, ok := conn.(syscall.Conn)
	if !ok {
		return nil
	}
	rawConn, err := sysConn.SyscallConn()
	if err != nil {
		return err
	}

	var sysErr error
	var n int
	err = rawConn.Read(func(fd uintptr) bool {
		// Go默认设置socket为非阻塞模式，调用syscall可以直接返回
		// 参考 Go源码：src/net/sock_cloexec.go下的sysSocket()函数
		n, sysErr = syscall.Read(int(fd), buf)
		// 返回值为true可以不执行net库封装的阻塞和等待,直接返回
		return true
	})
	if err != nil {
		return err
	}

	// 连接已关闭, 返回io.EOF
	if n == 0 && sysErr == nil {
		report.ConnectionPoolRemoteEOF.Incr()
		return io.EOF
	}
	// 空闲的连接不应该读出数据，黏包处理错误
	if n > 0 {
		return errors.New("unexpected read from socket")
	}
	// 空闲连接正常状态返回EAGAIN或者EWOULDBLOCK
	if sysErr == syscall.EAGAIN || sysErr == syscall.EWOULDBLOCK {
		return nil
	}
	return sysErr
}
