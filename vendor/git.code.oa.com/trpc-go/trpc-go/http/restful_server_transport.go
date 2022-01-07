package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/restful"
	"git.code.oa.com/trpc-go/trpc-go/transport"
	reuseport "git.woa.com/trpc-go/go_reuseport"
	"github.com/valyala/fasthttp"
)

var (
	errReplaceRouter = errors.New("not allow to replace router when is based on fasthttp")
)

func init() {
	// 兼容 thttp
	restful.SetCtxForCompatibility(func(ctx context.Context, w http.ResponseWriter,
		r *http.Request) context.Context {
		return WithHeader(ctx, &Header{Response: w, Request: r})
	})
	transport.RegisterServerTransport("restful", DefaultRESTServerTransport)
}

// DefaultRESTServerTransport 默认 RESTful ServerTransport
var DefaultRESTServerTransport = NewRESTServerTransport(false, transport.WithReusePort(true))

// RESTServerTransport RESTful ServerTransport
type RESTServerTransport struct {
	basedOnFastHTTP bool
	opts            *transport.ServerTransportOptions
}

// NewRESTServerTransport 创建一个 RESTful ServerTransport
func NewRESTServerTransport(basedOnFastHTTP bool, opt ...transport.ServerTransportOption) transport.ServerTransport {
	opts := &transport.ServerTransportOptions{
		IdleTimeout: time.Minute,
	}

	for _, o := range opt {
		o(opts)
	}

	return &RESTServerTransport{
		basedOnFastHTTP: basedOnFastHTTP,
		opts:            opts,
	}
}

// ListenAndServe 实现 transport.ServerTransport 接口
func (st *RESTServerTransport) ListenAndServe(ctx context.Context, opt ...transport.ListenServeOption) error {
	opts := &transport.ListenServeOptions{
		Network: "tcp",
	}

	for _, o := range opt {
		o(opts)
	}

	// 获取 listener
	ln, err := st.getListener(opts)
	if err != nil {
		return err
	}
	// 保存 listener
	if err := transport.SaveListener(ln); err != nil {
		return fmt.Errorf("save restful listener error: %w", err)
	}
	// 转成 tcpKeepAliveListener
	if tcpln, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{tcpln}
	}
	// 配置 tls
	if len(opts.TLSKeyFile) != 0 && len(opts.TLSCertFile) != 0 {
		tlsConf, err := generateTLSConfig(opts)
		if err != nil {
			return err
		}
		ln = tls.NewListener(ln, tlsConf)
	}

	return st.serve(ctx, ln, opts)
}

// serve 开启服务
func (st *RESTServerTransport) serve(ctx context.Context, ln net.Listener,
	opts *transport.ListenServeOptions) error {
	// 获取 router
	router := restful.GetRouter(opts.ServiceName)
	if router == nil {
		return fmt.Errorf("service %s router not registered", opts.ServiceName)
	}

	if st.basedOnFastHTTP { // 基于 fasthttp
		r, ok := router.(*restful.Router)
		if !ok {
			return errReplaceRouter
		}
		server := &fasthttp.Server{Handler: r.HandleRequestCtx}
		go func() {
			_ = server.Serve(ln)
		}()
		if st.opts.ReusePort {
			go func() {
				<-ctx.Done()
				_ = server.Shutdown()
			}()
		}
	} else { // 基于 golang net/http
		server := &http.Server{Addr: opts.Address, Handler: router}
		go func() {
			_ = server.Serve(ln)
		}()
		if st.opts.ReusePort {
			go func() {
				<-ctx.Done()
				_ = server.Shutdown(context.TODO())
			}()
		}
	}

	return nil
}

// getListener 获取 listener
func (st *RESTServerTransport) getListener(opts *transport.ListenServeOptions) (net.Listener, error) {
	var err error
	var ln net.Listener

	v, _ := os.LookupEnv(transport.EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		pln, err := transport.GetPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}

		ln, ok = pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}

		return ln, nil
	}

	if st.opts.ReusePort {
		ln, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("restful reuseport listen error: %w", err)
		}
	} else {
		ln, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("restful listen error: %w", err)
		}
	}

	return ln, nil
}

// generateTLSConfig 生成 tls 配置
func generateTLSConfig(opts *transport.ListenServeOptions) (*tls.Config, error) {
	tlsConf := &tls.Config{}

	cert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	tlsConf.Certificates = []tls.Certificate{cert}

	// 双向认证
	if opts.CACertFile != "" {
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		if opts.CACertFile != "root" {
			ca, err := ioutil.ReadFile(opts.CACertFile)
			if err != nil {
				return nil, err
			}
			pool := x509.NewCertPool()
			ok := pool.AppendCertsFromPEM(ca)
			if !ok {
				return nil, errors.New("failed to append certs from pem")
			}
			tlsConf.ClientCAs = pool
		}
	}

	return tlsConf, nil
}
