// +build windows

// Copyright (C) 2017 Ma Weiwei, Max Riveiro
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package reuseport

import (
	"net"
	"syscall"
)

var ListenerBacklogMaxSize = maxListenerBacklog()

func maxListenerBacklog() int {
	return syscall.SOMAXCONN
}

func NewReusablePortListener(proto, addr string) (net.Listener, error) {
	return net.Listen(proto, addr)
}

func NewReusablePortPacketConn(proto, addr string) (net.PacketConn, error) {
	return net.ListenPacket(proto, addr)
}

// Listen function is an alias for NewReusablePortListener.
func Listen(proto, addr string) (l net.Listener, err error) {
	return NewReusablePortListener(proto, addr)
}

// ListenPacket is an alias for NewReusablePortPacketConn.
func ListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return NewReusablePortPacketConn(proto, addr)
}
