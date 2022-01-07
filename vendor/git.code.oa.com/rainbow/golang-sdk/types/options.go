package types

import (
	"time"

	"git.code.oa.com/rainbow/golang-sdk/log"
)

var (
	defaultFileCachePath = "/tmp/configFile.dump"
)

// InitOptions initialization option
type InitOptions struct {
	// ConnectStr 配置中心地址
	ConnectStr string

	// IsConnectLocalAgent 是否连接本地配置中心agent
	IsConnectLocalAgent bool

	// IsUsingLocalCache 是否开启本地内存缓存
	IsUsingLocalCache bool

	// IsUsingFileCache 是否开启sdk文件缓存
	IsUsingFileCache bool

	// FileCachePath sdk文件缓存路径，默认值defaultFileCachePath
	FileCachePath string

	// TimeoutCS  请求Config_service超时时间
	TimeoutCS time.Duration

	// TimeoutPolling 长轮询等待时间
	TimeoutPolling time.Duration

	// RemoteProto 请求远端配置中心协议(trpc/http)
	RemoteProto string

	// AppID 与Groups配合进行预拉取group
	AppID string
	// Groups 不为空，将预拉取这些group
	Groups []string
	// EnvName 环境名称
	EnvName string

	// UserID 用户ID
	UserID string
	//UserKey 用户key
	UserKey string
	// HmacWay 签名方式:sha256或sha1 默认sha1
	HmacWay string
	// OpenSign 开启签名
	OpenSign bool
	// Heartbeat 心跳包间隔
	Heartbeat time.Duration
	// LogName 日志文件
	LogName string
	// LogLevel 日志等级
	LogLevel log.LogLevel

	// TimeoutNaming 寻址超时时间
	TimeoutNaming time.Duration

	// TimeoutDownload 文件下载超时时间默认5 Minute
	TimeoutDownload time.Duration

	// TimeoutConnNaming 寻址
	TimeoutConnNaming time.Duration
}

// AssignInitOption 设置InitOption
type AssignInitOption func(*InitOptions)

// NewInitOptions 参数设置
func NewInitOptions(opts ...AssignInitOption) InitOptions {
	var options InitOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.FileCachePath == "" {
		options.FileCachePath = defaultFileCachePath
	}

	if options.TimeoutCS == time.Duration(0) {
		options.TimeoutCS = 3 * time.Second
	}

	if options.TimeoutPolling == time.Duration(0) {
		options.TimeoutPolling = 60 * time.Second
	}

	if options.RemoteProto == "" {
		options.RemoteProto = "http"
	}
	if options.Heartbeat == time.Duration(0) {
		options.Heartbeat = 60 * time.Second
	}
	if options.HmacWay == "" {
		options.HmacWay = "sha1"
	}
	if options.LogName != "" {
		log.SetOutputByName(options.LogName)
		log.SetLevel(options.LogLevel)
	}
	if options.TimeoutNaming == time.Duration(0) {
		options.TimeoutNaming = 2 * time.Second
	}
	if options.TimeoutConnNaming == time.Duration(0) {
		options.TimeoutConnNaming = 2 * time.Second
	}
	if options.TimeoutDownload == time.Duration(0) {
		options.TimeoutDownload = 5 * time.Minute
	}
	log.SetRotateByDay()
	return options
}

// ConnectStr  connect string
func ConnectStr(connStr string) AssignInitOption {
	return func(o *InitOptions) {
		o.ConnectStr = connStr
	}
}

// IsConnectLocalAgent is connect local agent
func IsConnectLocalAgent(agent bool) AssignInitOption {
	return func(o *InitOptions) {
		o.IsConnectLocalAgent = agent
	}
}

// IsUsingLocalCache local cache
func IsUsingLocalCache(cache bool) AssignInitOption {
	return func(o *InitOptions) {
		o.IsUsingLocalCache = cache
	}
}

// IsUsingFileCache file cache
func IsUsingFileCache(cache bool) AssignInitOption {
	return func(o *InitOptions) {
		o.IsUsingFileCache = cache
	}
}

// FileCachePath cache path
func FileCachePath(path string) AssignInitOption {
	return func(o *InitOptions) {
		o.FileCachePath = path
	}
}

// TimeoutCS timeout
func TimeoutCS(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.TimeoutCS = d
	}
}

// Heartbeat  心跳
func Heartbeat(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.Heartbeat = d
	}
}

// TimeoutPolling timeout polling
func TimeoutPolling(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.TimeoutPolling = d
	}
}

// RemoteProto remote proto
func RemoteProto(proto string) AssignInitOption {
	return func(o *InitOptions) {
		o.RemoteProto = proto
	}
}

// AppID app id
func AppID(app string) AssignInitOption {
	return func(o *InitOptions) {
		o.AppID = app
	}
}

// EnvName 环境名称
func EnvName(env string) AssignInitOption {
	return func(o *InitOptions) {
		o.EnvName = env
	}
}

// Groups group preload
func Groups(group ...string) AssignInitOption {
	return func(o *InitOptions) {
		o.Groups = append(o.Groups, group...)
	}
}

// UserID user id
func UserID(userID string) AssignInitOption {
	return func(o *InitOptions) {
		o.UserID = userID
	}
}

// UserKey user key
func UserKey(userKey string) AssignInitOption {
	return func(o *InitOptions) {
		o.UserKey = userKey
	}
}

// HmacWay  hmac way
func HmacWay(hmacWay string) AssignInitOption {
	return func(o *InitOptions) {
		o.HmacWay = hmacWay
	}
}

// LogName 日志文件名称
func LogName(name string) AssignInitOption {
	return func(o *InitOptions) {
		o.LogName = name
	}
}

// LogLevel 日志文件名称
func LogLevel(level log.LogLevel) AssignInitOption {
	return func(o *InitOptions) {
		o.LogLevel = level
	}
}

// OpenSign  开启签名校验
func OpenSign(opn bool) AssignInitOption {
	return func(o *InitOptions) {
		o.OpenSign = opn
	}
}

// TimeoutNaming 寻址超时时间
func TimeoutNaming(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.TimeoutNaming = d
	}
}

// TimeoutDownload 文件下载超时时间
func TimeoutDownload(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.TimeoutDownload = d
	}
}

// TimeoutConnNaming 寻址连接超时时间
func TimeoutConnNaming(d time.Duration) AssignInitOption {
	return func(o *InitOptions) {
		o.TimeoutConnNaming = d
	}
}

// GetOptions options when get value
type GetOptions struct {
	Remote        bool // 不读缓存，直接从远程服务端获取
	NoUpdateCache bool // 该次请求不更新缓存
	DefaultValue  string
	AppID         string
	Group         string
	Version       string
	IP            string            // 客户端信息:IP
	Uin           string            // 客户端信息:Uin
	ClientInfo    map[string]string // 客户端信息，除了IP和UIN
	UserID        string            // 用户ID，签名时用到
	UserKey       string            // 用户key，签名时用到
	HmacWay       string            // 签名方式:sha256或sha1 默认sha1
	Start         int32             // table类型起始id, 必须是offset的整数倍
	Offset        int32             // 偏移的id值，必须是10的n次方且不大于100000，大于0表示，使用分页
	EnvName       string            // 环境名称
}

// AssignGetOption 设置GetOption
type AssignGetOption func(*GetOptions)

// NewGetOptions get 参数设置
func NewGetOptions(opts ...AssignGetOption) GetOptions {
	var options GetOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.HmacWay == "" {
		options.HmacWay = "sha1"
	}
	return options
}

// Reassign 重新赋值，新覆盖旧
func (o *GetOptions) Reassign(opts ...AssignGetOption) *GetOptions {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithDefault 设置默认值
func WithDefault(defaultVal string) AssignGetOption {
	return func(o *GetOptions) {
		o.DefaultValue = defaultVal
	}
}

// WithAppID 设置AppID
func WithAppID(id string) AssignGetOption {
	return func(o *GetOptions) {
		o.AppID = id
	}
}

// WithGroup 设置group 名字
func WithGroup(group string) AssignGetOption {
	return func(o *GetOptions) {
		o.Group = group
	}
}

// WithVersion 设置版本
func WithVersion(version string) AssignGetOption {
	return func(o *GetOptions) {
		o.Version = version
	}
}

// WithIP 设置本地IP
func WithIP(ip string) AssignGetOption {
	return func(o *GetOptions) {
		o.IP = ip
	}
}

// WithUin 设置uin
func WithUin(uin string) AssignGetOption {
	return func(o *GetOptions) {
		o.Uin = uin
	}
}

// WithRemote 支持从远程server获取(不读缓存）
func WithRemote(r bool) AssignGetOption {
	return func(o *GetOptions) {
		o.Remote = r
	}
}

// SetNoUpdateCache 该次请求不更新缓存, true 不更新，false更新，默认不更新
func SetNoUpdateCache(nc bool) AssignGetOption {
	return func(o *GetOptions) {
		o.NoUpdateCache = nc
	}
}

// AddClientInfo 添加用户自定义 客户端信息
func AddClientInfo(k, v string) AssignGetOption {
	return func(o *GetOptions) {
		if o.ClientInfo == nil {
			o.ClientInfo = make(map[string]string)
		}
		o.ClientInfo[k] = v
	}
}

// WithUserID user id
func WithUserID(userID string) AssignGetOption {
	return func(o *GetOptions) {
		o.UserID = userID
	}
}

// WithUserKey user key
func WithUserKey(userKey string) AssignGetOption {
	return func(o *GetOptions) {
		o.UserKey = userKey
	}
}

// WithHmacWay  hmac way
func WithHmacWay(hmacWay string) AssignGetOption {
	return func(o *GetOptions) {
		o.HmacWay = hmacWay
	}
}

// WithStart table分页起始位置
func WithStart(start int32) AssignGetOption {
	return func(o *GetOptions) {
		o.Start = start
	}
}

// WithOffset table分页偏移
func WithOffset(offset int32) AssignGetOption {
	return func(o *GetOptions) {
		o.Offset = offset
	}
}

// WithEnvName 环境名称
func WithEnvName(env string) AssignGetOption {
	return func(o *GetOptions) {
		o.EnvName = env
	}
}

// SimpleString 打印GetOptions 重点几个字段
func (o *GetOptions) SimpleString() string {
	return "AppID:" + o.AppID + "\t Group:" + o.Group + "\t Env:" + o.EnvName
}
