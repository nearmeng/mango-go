// Package http 默认支持http协议，提供http协议的rpc server，调用http协议的rpc client
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	stdhttp "net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/transport"
	"golang.org/x/net/http2"

	reuseport "git.woa.com/trpc-go/go_reuseport"
	valid "github.com/asaskevich/govalidator"
)

func init() {
	// 服务端 有协议文件服务 server transport
	transport.RegisterServerTransport("http", DefaultServerTransport)
	transport.RegisterServerTransport("http2", DefaultHTTP2ServerTransport)
	// 服务端 无协议文件服务 server transport
	transport.RegisterServerTransport("http_no_protocol", DefaultServerTransport)
	transport.RegisterServerTransport("http2_no_protocol", DefaultHTTP2ServerTransport)
	// 客户端 client transport
	transport.RegisterClientTransport("http", DefaultClientTransport)
	transport.RegisterClientTransport("http2", DefaultHTTP2ClientTransport)
}

// DefaultServerTransport 默认server http transport
var DefaultServerTransport = NewServerTransport(transport.WithReusePort(true))

// DefaultHTTP2ServerTransport 默认server http2 transport
var DefaultHTTP2ServerTransport = NewServerTransport(transport.WithReusePort(true))

// ServerTransport HTTP传输层
type ServerTransport struct {
	Server *stdhttp.Server // 支持外部配置
	opts   *transport.ServerTransportOptions
}

// NewServerTransport 创建http transport
//
// 默认空闲时间为1min，支持通过ServerTransportOption自定义
func NewServerTransport(opt ...transport.ServerTransportOption) transport.ServerTransport {
	opts := &transport.ServerTransportOptions{}

	// 将传入的func option写到opts字段中
	for _, o := range opt {
		o(opts)
	}
	s := &ServerTransport{
		opts: opts,
	}
	return s
}

// ListenAndServe 处理配置
func (t *ServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}
	for _, o := range opt {
		o(opts)
	}
	if opts.Handler == nil {
		return errors.New("http server transport handler empty")
	}
	return t.listenAndServeHTTP(ctx, opts)
}

var emptyBuf []byte

func (t *ServerTransport) listenAndServeHTTP(ctx context.Context, opts *transport.ListenServeOptions) error {
	// ServeHTTP http包注册的统一handle
	serveFunc := func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		m := &Header{Request: r, Response: w}
		innerCtx := WithHeader(r.Context(), m)

		// 生成新的空的通用消息结构数据，并保存到ctx里面
		innerCtx, msg := codec.WithNewMessage(innerCtx)
		defer codec.PutBackMessage(msg)

		// 记录LocalAddr和RemoteAddr到Context
		localAddr, ok := r.Context().Value(stdhttp.LocalAddrContextKey).(net.Addr)
		if ok {
			msg.WithLocalAddr(localAddr)
		}
		raddr, _ := net.ResolveTCPAddr("tcp", r.RemoteAddr)
		msg.WithRemoteAddr(raddr)
		_, err := opts.Handler.Handle(innerCtx, emptyBuf)
		if err != nil {
			log.Errorf("http server transport handle fail:%v", err)
			if err == ErrEncodeMissingHeader {
				w.WriteHeader(500)
			}
			return
		}
	}

	s, err := newHTTPServer(serveFunc, opts)
	if err != nil {
		return err
	}

	t.configureHTTPServer(s, opts)
	if err := t.serve(ctx, s, opts); err != nil {
		return err
	}
	return nil
}

func (t *ServerTransport) serve(ctx context.Context, s *stdhttp.Server, opts *transport.ListenServeOptions) error {
	ln, err := t.getListener(opts.Network, s.Addr)
	if err != nil {
		return err
	}

	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save http listener error: %v", err)
	}

	// 端口重用，内核分发IO ReadReady事件到多核多线程，加速IO效率
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		go func() {
			_ = s.ServeTLS(tcpKeepAliveListener{ln.(*net.TCPListener)}, opts.TLSCertFile, opts.TLSKeyFile)
		}()
	} else {
		go func() {
			_ = s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
		}()
	}

	if t.opts.ReusePort {
		go func() {
			<-ctx.Done()
			_ = s.Shutdown(context.TODO())
		}()
		return nil
	}
	return nil
}

func (t *ServerTransport) getListener(network, addr string) (net.Listener, error) {
	var ln net.Listener
	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		pln, err := transport.GetPassedListener(network, addr)
		if err != nil {
			return nil, err
		}

		ln, ok = pln.(net.Listener)
		if !ok {
			return nil, fmt.Errorf("invalid net.Listener")
		}

		return ln, nil
	}

	if t.opts.ReusePort {
		ln, err := reuseport.Listen(network, addr)
		if err != nil {
			return nil, fmt.Errorf("http reuseport listen error:%v", err)
		}
		return ln, nil
	}

	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("http listen error:%v", err)
	}

	return ln, nil
}

// configureHTTPServer 设置http server相关属性
func (t *ServerTransport) configureHTTPServer(svr *stdhttp.Server, opts *transport.ListenServeOptions) {
	if t.Server != nil {
		svr.ReadTimeout = t.Server.ReadTimeout
		svr.ReadHeaderTimeout = t.Server.ReadHeaderTimeout
		svr.WriteTimeout = t.Server.WriteTimeout
		svr.MaxHeaderBytes = t.Server.MaxHeaderBytes
		svr.IdleTimeout = t.Server.IdleTimeout
		svr.ConnState = t.Server.ConnState
		svr.ErrorLog = t.Server.ErrorLog
	}

	idleTimeout := opts.IdleTimeout
	if t.opts.IdleTimeout > 0 {
		idleTimeout = t.opts.IdleTimeout
	}
	svr.IdleTimeout = idleTimeout
}

// newHttpServer 新建 server
func newHTTPServer(serveFunc func(w stdhttp.ResponseWriter, r *stdhttp.Request),
	opts *transport.ListenServeOptions) (*stdhttp.Server, error) {
	s := &stdhttp.Server{
		Addr:    opts.Address,
		Handler: stdhttp.HandlerFunc(serveFunc),
	}
	if len(opts.CACertFile) != 0 { // 开启双向认证，验证客户端证书
		s.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		certPool, err := getCertPool(opts.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("http server get ca cert file error:%v", err)
		}
		s.TLSConfig.ClientCAs = certPool
	}
	return s, nil
}

// getCertPool 获取证书信息
func getCertPool(caCertFile string) (*x509.CertPool, error) {
	if caCertFile != "root" { // root 表示使用本机安装的root ca证书验证client，不是root表示使用输入ca证书验证client
		ca, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		ok := pool.AppendCertsFromPEM(ca)
		if !ok {
			return nil, fmt.Errorf("appendCertsFromPEM fail")
		}
		return pool, nil
	}
	return nil, nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept accept 新请求
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ClientTransport client端http transport
type ClientTransport struct {
	opts           *transport.ClientTransportOptions
	stdhttp.Client                            // http client，公开变量，允许用户自定义设置
	tlsClients     map[string]*stdhttp.Client // 不同的证书文件 不同的tls client
	tlsLock        sync.RWMutex
	http2Only      bool
}

// DefaultClientTransport 默认client http transport
var DefaultClientTransport = NewClientTransport(false)

// DefaultHTTP2ClientTransport 默认client http2 transport
var DefaultHTTP2ClientTransport = NewClientTransport(true)

// DefaultClientMaxIdleConnsPerHost 每个host的连接池中最大空闲链接数
var DefaultClientMaxIdleConnsPerHost = 100

// NewClientTransport 创建http transport
func NewClientTransport(http2Only bool, opt ...transport.ClientTransportOption) transport.ClientTransport {
	opts := &transport.ClientTransportOptions{}

	// 将传入的func option写到opts字段中
	for _, o := range opt {
		o(opts)
	}
	return &ClientTransport{
		opts: opts,
		Client: stdhttp.Client{
			Transport: NewRoundTripper(StdHTTPTransport),
		},
		tlsClients: make(map[string]*stdhttp.Client),
		http2Only:  http2Only,
	}
}

// setReqHeader 设置 ClientReqHeader
func (ct *ClientTransport) setReqHeader(reqHeader *ClientReqHeader,
	reqbody []byte, msg codec.Msg, opts *transport.RoundTripOptions) error {
	if len(reqHeader.Schema) == 0 {
		if len(opts.CACertFile) > 0 || strings.HasSuffix(opts.Address, ":443") {
			reqHeader.Schema = "https"
		} else {
			reqHeader.Schema = "http"
		}
	}

	// 没有设置request，框架自动生成，除非用户自己NewRequest
	if reqHeader.Request == nil {
		url := fmt.Sprintf("%s://%s%s", reqHeader.Schema, opts.Address, msg.ClientRPCName())
		if !valid.IsURL(url) {
			return errs.NewFrameError(errs.RetClientValidateFail,
				"http client invalid url: "+url)
		}

		request, err := stdhttp.NewRequest(reqHeader.Method, url, bytes.NewBuffer(reqbody))
		if err != nil {
			return errs.NewFrameError(errs.RetClientNetErr,
				"http client transport NewRequest: "+err.Error())
		}

		reqHeader.Request = request
		if reqHeader.Header != nil {
			reqHeader.Request.Header = make(stdhttp.Header)
			for h, val := range reqHeader.Header {
				reqHeader.Request.Header[h] = val
			}
		}
		if len(reqHeader.Host) != 0 {
			reqHeader.Request.Host = reqHeader.Host
		}
		reqHeader.Request.Header.Set(TrpcCaller, msg.CallerServiceName())
		reqHeader.Request.Header.Set(TrpcCallee, msg.CalleeServiceName())
		reqHeader.Request.Header.Set(TrpcTimeout, strconv.Itoa(int(msg.RequestTimeout()/time.Millisecond)))
		if msg.CompressType() > 0 {
			reqHeader.Request.Header.Set("Content-Encoding", compressTypeContentEncoding[msg.CompressType()])
		}
		if msg.SerializationType() != codec.SerializationTypeNoop {
			if len(reqHeader.Request.Header.Get("Content-Type")) == 0 {
				reqHeader.Request.Header.Set("Content-Type",
					serializationTypeContentType[msg.SerializationType()])
			}
		}
	}

	if len(msg.ClientMetaData()) > 0 {
		m := make(map[string]string)
		for k, v := range msg.ClientMetaData() {
			m[k] = base64.StdEncoding.EncodeToString(v)
		}
		// 设置染色信息
		if msg.Dyeing() {
			m[TrpcDyeingKey] = base64.StdEncoding.EncodeToString([]byte(msg.DyeingKey()))
			reqHeader.Request.Header.Set(TrpcMessageType,
				strconv.Itoa(int(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)))
		}
		m[TrpcEnv] = base64.StdEncoding.EncodeToString([]byte(msg.EnvTransfer()))
		val, _ := codec.Marshal(codec.SerializationTypeJSON, m)
		if reqHeader.Request.Header == nil {
			reqHeader.Request.Header = make(stdhttp.Header)
		}
		reqHeader.Request.Header.Set(TrpcTransInfo, string(val))
	}
	if len(opts.TLSServerName) == 0 {
		opts.TLSServerName = reqHeader.Request.Host
	}
	return nil
}

// RoundTrip 收发http包, 回包http response放到ctx里面，这里不需要返回rspbuf
func (ct *ClientTransport) RoundTrip(ctx context.Context, reqbody []byte,
	callOpts ...transport.RoundTripOption) (rspbody []byte, err error) {
	msg := codec.Message(ctx)
	reqHeader, ok := msg.ClientReqHead().(*ClientReqHeader)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"http client transport: ReqHead should be type of *http.ClientReqHeader")
	}
	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, errs.NewFrameError(errs.RetClientEncodeFail,
			"http client transport: RspHead should be type of *http.ClientRspHeader")
	}

	var opts transport.RoundTripOptions
	for _, o := range callOpts {
		o(&opts)
	}

	// 设置 reqHeader
	if err := ct.setReqHeader(reqHeader, reqbody, msg, &opts); err != nil {
		return nil, err
	}
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			msg.WithRemoteAddr(connInfo.Conn.RemoteAddr())
		},
	}
	request := reqHeader.Request.WithContext(httptrace.WithClientTrace(ctx, trace))

	client, err := ct.getStdHTTPClient(opts.CACertFile, opts.TLSCertFile,
		opts.TLSKeyFile, opts.TLSServerName)
	if err != nil {
		return nil, err
	}

	rspHeader.Response, err = client.Do(request)
	if err != nil {
		if e, ok := err.(*url.Error); ok {
			if e.Timeout() {
				return nil, errs.NewFrameError(errs.RetClientTimeout,
					"http client transport RoundTrip timeout: "+err.Error())
			}
		}
		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"http client transport RoundTrip canceled: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"http client transport RoundTrip: "+err.Error())
	}
	return emptyBuf, nil
}

func (ct *ClientTransport) getTLSConfig(caFile, certFile, keyFile, serverName string) (*tls.Config, error) {
	conf := &tls.Config{}
	if caFile == "none" {
		// 忽略校验server证书
		conf.InsecureSkipVerify = true
		return conf, nil
	}
	// 验证server服务名
	conf.ServerName = serverName
	certPool, err := getCertPool(caFile)
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientDecodeFail,
			"http client transport getTLSConfig get ca file error: "+err.Error())
	}
	conf.RootCAs = certPool
	if len(certFile) != 0 {
		// https服务开启双向认证，需要传送client自身证书给server
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientDecodeFail,
				"http client transport getTLSConfig load cert file error: "+err.Error())
		}
		conf.Certificates = []tls.Certificate{cert}
	}

	return conf, nil
}

func (ct *ClientTransport) getStdHTTPClient(caFile, certFile,
	keyFile, serverName string) (*stdhttp.Client, error) {
	if len(caFile) == 0 { // http请求，共用一个client
		return &ct.Client, nil
	}

	cacheKey := fmt.Sprintf("%s-%s-%s", caFile, certFile, serverName)
	ct.tlsLock.RLock()
	cli, ok := ct.tlsClients[cacheKey]
	ct.tlsLock.RUnlock()
	if ok {
		return cli, nil
	}

	ct.tlsLock.Lock()
	defer ct.tlsLock.Unlock()
	cli, ok = ct.tlsClients[cacheKey]
	if ok {
		return cli, nil
	}

	conf, err := ct.getTLSConfig(caFile, certFile, keyFile, serverName)
	if err != nil {
		return nil, err
	}

	client := &stdhttp.Client{
		CheckRedirect: ct.Client.CheckRedirect,
		Timeout:       ct.Client.Timeout,
	}
	if ct.http2Only {
		client.Transport = &http2.Transport{
			TLSClientConfig: conf,
		}
	} else {
		tr := StdHTTPTransport.Clone()
		tr.TLSClientConfig = conf
		client.Transport = NewRoundTripper(tr)
	}
	ct.tlsClients[cacheKey] = client
	return client, nil
}

// StdHTTPTransport 所有http&https使用的RoundTripper对象
var StdHTTPTransport = &stdhttp.Transport{
	Proxy: stdhttp.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          200,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	MaxIdleConnsPerHost:   100,
	MaxConnsPerHost:       200,
	ExpectContinueTimeout: time.Second,
}

// NewRoundTripper 创建新的NewRoundTripper 可由业务替换
var NewRoundTripper = newValueDetachedTransport
