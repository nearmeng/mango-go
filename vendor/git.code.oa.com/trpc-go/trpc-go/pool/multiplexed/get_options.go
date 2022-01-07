package multiplexed

import "git.code.oa.com/trpc-go/trpc-go/codec"

// GetOptions get conn configuration
type GetOptions struct {
	FramerBuilder codec.FramerBuilder
	Msg           codec.Msg

	CACertFile    string // ca证书
	TLSCertFile   string // client证书
	TLSKeyFile    string // client秘钥
	TLSServerName string // client校验server的服务名, 不填时默认为http的hostname

	LocalAddr string
}

// NewGetOptions 创建GetOptions
func NewGetOptions() GetOptions {
	return GetOptions{}
}

// WithFramerBuilder 设置 FramerBuilder
func (o *GetOptions) WithFramerBuilder(fb codec.FramerBuilder) {
	o.FramerBuilder = fb
}

// WithDialTLS 设置client支持TLS
func (o *GetOptions) WithDialTLS(certFile, keyFile, caFile, serverName string) {
	o.TLSCertFile = certFile
	o.TLSKeyFile = keyFile
	o.CACertFile = caFile
	o.TLSServerName = serverName
}

// WithMsg 设置 Msg
func (o *GetOptions) WithMsg(msg codec.Msg) {
	o.Msg = msg
}

// WithLocalAddr 建立连接时指定本地地址，多网卡时默认随机选择
func (o *GetOptions) WithLocalAddr(addr string) {
	o.LocalAddr = addr
}
