// Package admin 实现了一些常用的管理功能
package admin

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/config"
	"git.code.oa.com/trpc-go/trpc-go/log"

	jsoniter "github.com/json-iterator/go"
)

var (
	pattenCmds     = "/cmds"
	pattenVersion  = "/version"
	pattenLoglevel = "/cmds/loglevel"
	pattenConfig   = "/cmds/config"

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

// return param
var (
	ReturnErrCodeParam = "errorcode"
	ReturnMessageParam = "message"
	ErrCodeServer      = 1
)

// TrpcAdminServer 管理服务，实现server.Service
type TrpcAdminServer struct {
	config    *adminConfig
	server    *http.Server
	closeOnce sync.Once
	closeErr  error
	router    Router
}

// NewTrpcAdminServer 创建一个新的AdminServer
func NewTrpcAdminServer(opts ...Option) *TrpcAdminServer {
	cfg := loadDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	s := &TrpcAdminServer{
		config: cfg,
	}
	s.initRouter()
	return s
}

// 内部router
var defaultRouter = NewRouter()

// 保证初始化一次 defaultRouter
var once sync.Once

// 初始化
func (s *TrpcAdminServer) initRouter() {
	once.Do(func() {
		defaultRouter.Config(pattenCmds, s.handleCmds).Desc("管理命令列表")
		defaultRouter.Config(pattenVersion, s.handleVersion).Desc("框架版本")
		defaultRouter.Config(pattenLoglevel, s.handleLogLevel).Desc("查看/设置框架的日志级别")
		defaultRouter.Config(pattenConfig, s.handleConfig).Desc("查看框架配置文件")

		defaultRouter.Config("/debug/pprof/", pprof.Index)
		defaultRouter.Config("/debug/pprof/cmdline", pprof.Cmdline)
		defaultRouter.Config("/debug/pprof/profile", pprof.Profile)
		defaultRouter.Config("/debug/pprof/symbol", pprof.Symbol)
		defaultRouter.Config("/debug/pprof/trace", pprof.Trace)

		// 删除 http.DefaultServeMux 注册的 pprof 路由，避免引起安全问题：https://github.com/golang/go/issues/22085
		err := unregisterHandlers([]string{
			"/debug/pprof/",
			"/debug/pprof/cmdline",
			"/debug/pprof/profile",
			"/debug/pprof/symbol",
			"/debug/pprof/trace",
		})
		if err != nil {
			log.Errorf("failed to unregister pprof handlers from http.DefaultServeMux, err: %v", err)
		}
		s.router = defaultRouter
	})
}

// Register 实现server.Service
func (s *TrpcAdminServer) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	// 返回nil， server.Server.Register会把所有业务实现接口注册到所有service里面(也会调用TrpcAdminServer.Register)
	return nil
}

// Serve 启动http Server
func (s *TrpcAdminServer) Serve() error {
	cfg := s.config
	if cfg.enableTLS {
		return errors.New("not support yet")
	}

	s.server = &http.Server{
		Addr:         cfg.getAddr(),
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
		Handler:      s.router,
	}

	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close 关闭服务
func (s *TrpcAdminServer) Close(ch chan struct{}) error {
	pid := os.Getpid()
	s.closeOnce.Do(s.close)
	log.Infof("process:%d, admin server, closed", pid)
	if ch != nil {
		ch <- struct{}{}
	}
	return s.closeErr
}

// HandleFunc 注册自定义服务接口
func HandleFunc(patten string, handler func(w http.ResponseWriter, r *http.Request)) *RouterHandler {
	return defaultRouter.Config(patten, handler)
}

func (s *TrpcAdminServer) close() {
	if s.server == nil {
		return
	}
	s.closeErr = s.server.Close()
}

// ErrorOutput 统一错误输出
func ErrorOutput(w http.ResponseWriter, error string, code int) {
	var ret = getDefaultRes()
	ret[ReturnErrCodeParam] = code
	ret[ReturnMessageParam] = error
	_ = json.NewEncoder(w).Encode(ret)
}

// handleCmds 管理命令列表
func (s *TrpcAdminServer) handleCmds(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var cmds []string
	list := s.router.List()
	for _, item := range list {
		cmds = append(cmds, item.GetPatten())
	}
	var ret = getDefaultRes()
	ret["cmds"] = cmds

	_ = json.NewEncoder(w).Encode(ret)
}

// getDefaultRes admin默认输出格式
func getDefaultRes() map[string]interface{} {
	return map[string]interface{}{
		ReturnErrCodeParam: 0,
		ReturnMessageParam: "",
	}
}

// handleVersion 版本号查询
func (s *TrpcAdminServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	ret := map[string]interface{}{
		ReturnErrCodeParam: 0,
		ReturnMessageParam: "",
		"version":          s.config.version,
	}
	_ = json.NewEncoder(w).Encode(ret)
}

// getLevel 获取logger对象output流等级
func getLevel(logger log.Logger, output string) string {
	level := logger.GetLevel(output)
	return log.LevelStrings[level]
}

// handleLogLevel 查询设置日志等级
func (s *TrpcAdminServer) handleLogLevel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := r.ParseForm(); err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	name := r.Form.Get("logger")
	if name == "" {
		name = "default"
	}
	output := r.Form.Get("output")
	if output == "" {
		output = "0" // 没有ouput，默认为第一个output，一般用户也只配置一个
	}

	logger := log.Get(name)
	if logger == nil {
		ErrorOutput(w, "logger not found", ErrCodeServer)
		return
	}

	var ret = getDefaultRes()
	if r.Method == http.MethodGet {
		ret["level"] = getLevel(logger, output)
		_ = json.NewEncoder(w).Encode(ret)
	} else if r.Method == http.MethodPut {
		level := r.PostForm.Get("value")

		ret["prelevel"] = getLevel(logger, output)
		logger.SetLevel(output, log.LevelNames[level])
		ret["level"] = getLevel(logger, output)

		_ = json.NewEncoder(w).Encode(ret)
	}
}

// handleConfig 配置文件内容查询
func (s *TrpcAdminServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	buf, err := ioutil.ReadFile(s.config.configPath)
	if err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	unmarshaler := config.GetUnmarshaler("yaml")
	if unmarshaler == nil {
		ErrorOutput(w, "cannot find yaml unmarshaler", ErrCodeServer)
		return
	}

	conf := map[interface{}]interface{}{}
	if err = unmarshaler.Unmarshal(buf, &conf); err != nil {
		ErrorOutput(w, err.Error(), ErrCodeServer)
		return
	}

	var ret = getDefaultRes()
	ret["content"] = conf

	_ = json.NewEncoder(w).Encode(ret)
}
