package http

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	stdhttp "net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
)

const (
	// TrpcVersion 版本
	TrpcVersion = "trpc-version"
	// TrpcCallType 调用类型
	TrpcCallType = "trpc-call-type"
	// TrpcMessageType 消息类型
	TrpcMessageType = "trpc-message-type"
	// TrpcRequestID 请求id
	TrpcRequestID = "trpc-request-id"
	// TrpcTimeout 请求超时
	TrpcTimeout = "trpc-timeout"
	// TrpcCaller 主调方
	TrpcCaller = "trpc-caller"
	// TrpcCallee 被调方
	TrpcCallee = "trpc-callee"
	// TrpcTransInfo 透传信息
	TrpcTransInfo = "trpc-trans-info"
	// TrpcEnv 透传环境key
	TrpcEnv = "trpc-env"
	// TrpcDyeingKey 染色key
	TrpcDyeingKey = "trpc-dyeing-key"
)

var contentTypeSerializationType = map[string]int{
	"application/json":                  codec.SerializationTypeJSON,
	"application/protobuf":              codec.SerializationTypePB,
	"application/x-protobuf":            codec.SerializationTypePB,
	"application/pb":                    codec.SerializationTypePB,
	"application/proto":                 codec.SerializationTypePB,
	"application/jce":                   codec.SerializationTypeJCE,
	"application/flatbuffer":            codec.SerializationTypeFlatBuffer,
	"application/octet-stream":          codec.SerializationTypeNoop,
	"application/x-www-form-urlencoded": codec.SerializationTypeForm,
	"application/xml":                   codec.SerializationTypeXML,
	"multipart/form-data":               codec.SerializationTypeFormData,
}

var serializationTypeContentType = map[int]string{
	codec.SerializationTypeJSON:       "application/json",
	codec.SerializationTypePB:         "application/proto",
	codec.SerializationTypeJCE:        "application/jce",
	codec.SerializationTypeFlatBuffer: "application/flatbuffer",
	codec.SerializationTypeNoop:       "application/octet-stream",
	codec.SerializationTypeForm:       "application/x-www-form-urlencoded",
	codec.SerializationTypeXML:        "application/xml",
	codec.SerializationTypeFormData:   "multipart/form-data",
}

var contentEncodingCompressType = map[string]int{
	"gzip": codec.CompressTypeGzip,
}

var compressTypeContentEncoding = map[int]string{
	codec.CompressTypeGzip: "gzip",
}

// RegisterSerializer 注册新的自定义序列化方式,如 RegisterSerializer("text/plain", 130, xxxSerializer)
func RegisterSerializer(httpContentType string, serializationType int, serializer codec.Serializer) {
	codec.RegisterSerializer(serializationType, serializer)
	RegisterContentType(httpContentType, serializationType)
}

// RegisterContentType 注册已有的序列化方式，双向映射，如 RegisterContentType("text/plain", codec.SerializationTypeJSON)
func RegisterContentType(httpContentType string, serializationType int) {
	contentTypeSerializationType[httpContentType] = serializationType
	serializationTypeContentType[serializationType] = httpContentType
}

// SetContentType 设置单向映射关系，用于兼容老框架服务， 允许多个http content type映射到相同的trpc serialization type，
// 告诉框架使用何种序列化方式来解析这个content-type
// 如有些不规范的http server，返回content type是text/html,但实际上是json数据，
// 这个时候就可以这样设置：SetContentType("text/html", codec.SerializationTypeJSON)
func SetContentType(httpContentType string, serializationType int) {
	contentTypeSerializationType[httpContentType] = serializationType
}

// RegisterContentEncoding 注册已有的解压缩方式，如 RegisterContentEncoding("gzip", codec.CompressTypeGzip)
func RegisterContentEncoding(httpContentEncoding string, compressType int) {
	contentEncodingCompressType[httpContentEncoding] = compressType
	compressTypeContentEncoding[compressType] = httpContentEncoding
}

// RegisterStatus 注册trpc ret code到http status
func RegisterStatus(code int32, httpStatus int) {
	ErrsToHTTPStatus[code] = httpStatus
}

func init() {
	codec.Register("http", DefaultServerCodec, DefaultClientCodec)
	codec.Register("http2", DefaultServerCodec, DefaultClientCodec)
	// 支持无协议文件自定义路由,特性隔离,无协议文件服务可和有协议文件服务同时存在
	codec.Register("http_no_protocol", DefaultNoProtocolServerCodec, DefaultClientCodec)
	codec.Register("http2_no_protocol", DefaultNoProtocolServerCodec, DefaultClientCodec)
}

var (
	// DefaultClientCodec is default http client codec
	DefaultClientCodec = &ClientCodec{}

	// DefaultServerCodec is default http server codec
	DefaultServerCodec = &ServerCodec{
		AutoGenTrpcHead: true,
		ErrHandler:      defaultErrHandler,
		RspHandler:      defaultRspHandler,
		AutoReadBody:    true,
	}

	// DefaultNoProtocolServerCodec is default http no protocol server codec
	DefaultNoProtocolServerCodec = &ServerCodec{
		AutoGenTrpcHead: true,
		ErrHandler:      defaultErrHandler,
		RspHandler:      defaultRspHandler,
		AutoReadBody:    false,
	}
)

// ErrEncodeMissingHeader ctx丢失了header信息，定义错误用以在transport中特殊处理
var ErrEncodeMissingHeader = errors.New("trpc/http: server encode missing http header in context")

// ServerCodec http Server端解码器
type ServerCodec struct {
	// AutoGenTrpcHead 自动转换trpc头，
	// 业务可通过 http.DefaultServerCodec.AutoGenTrpcHead = true 设置是否自动转换
	AutoGenTrpcHead bool

	// ErrHandler 错误码处理函数，默认填充到header里面，
	// 业务可通过 http.DefaultServerCodec.ErrHandler = func(rsp, req, err) {} 替换
	ErrHandler ErrorHandler

	// RspHandler 返回数据处理函数，默认将数据直接返回，业务可定制此方法来塑型返回数据
	// 业务可通过 http.DefaultServerCodec.RspHandler = func(rsp, req, rspbody) {} 替换
	RspHandler ResponseHandler

	// AutoReadBody 自动读取http request body
	AutoReadBody bool
}

// ContextKey 定义http的contextkey
type ContextKey string

const (
	// ContextKeyHeader key of http header
	ContextKeyHeader = ContextKey("TRPC_SERVER_HTTP_HEADER")
	// ParseMultipartFormMaxMemory 解析请求体的最大内存 默认 32M
	ParseMultipartFormMaxMemory int64 = 32 << 20
)

// Header 封装http上下文
type Header struct {
	ReqBody  []byte
	Request  *stdhttp.Request
	Response stdhttp.ResponseWriter
}

// ClientReqHeader 封装http client请求的上下文
// 禁止在 NewClientProxy 等 Client 初始化时设置 ClientReqHeader
// ClientReqHeader 设置需要在每次调用时
type ClientReqHeader struct {
	Schema  string // http https
	Method  string
	Host    string
	Request *stdhttp.Request
	Header  stdhttp.Header
}

// AddHeader 添加http header
func (h *ClientReqHeader) AddHeader(key string, value string) {
	if h.Header == nil {
		h.Header = make(stdhttp.Header)
	}
	h.Header.Add(key, value)
}

// ClientRspHeader 封装http client请求响应的上下文
type ClientRspHeader struct {
	Response *stdhttp.Response
}

// ErrsToHTTPStatus 从框架errs retcode 映射到http status code
var ErrsToHTTPStatus = map[int32]int{
	errs.RetServerNoFunc:     404,
	errs.RetServerNoService:  404,
	errs.RetServerDecodeFail: 400,
	errs.RetServerEncodeFail: 500,
	errs.RetServerSystemErr:  500,
}

// Head 从context里获取对应的http header
func Head(ctx context.Context) *Header {
	if ret, ok := ctx.Value(ContextKeyHeader).(*Header); ok {
		return ret
	}
	return nil
}

// Request 从context里获取对应的http request
func Request(ctx context.Context) *stdhttp.Request {
	head := Head(ctx)
	if head == nil {
		return nil
	}
	return head.Request
}

// Response 从context里获取对应的http response
func Response(ctx context.Context) stdhttp.ResponseWriter {
	head := Head(ctx)
	if head == nil {
		return nil
	}
	return head.Response
}

// WithHeader 在context中设置http header
func WithHeader(ctx context.Context, value *Header) context.Context {
	return context.WithValue(ctx, ContextKeyHeader, value)
}

// setReqHeader 设置请求头
func (sc *ServerCodec) setReqHeader(head *Header, msg codec.Msg) error {
	if !sc.AutoGenTrpcHead { // 自动生成trpc head
		return nil
	}

	trpcReq := &trpc.RequestProtocol{}
	msg.WithServerReqHead(trpcReq)
	msg.WithServerRspHead(trpcReq)

	trpcReq.Func = []byte(msg.ServerRPCName())
	trpcReq.ContentType = uint32(msg.SerializationType())
	trpcReq.ContentEncoding = uint32(msg.CompressType())

	if v := head.Request.Header.Get(TrpcVersion); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Version = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcCallType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.CallType = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcMessageType); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.MessageType = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcRequestID); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.RequestId = uint32(i)
	}
	if v := head.Request.Header.Get(TrpcTimeout); v != "" {
		i, _ := strconv.Atoi(v)
		trpcReq.Timeout = uint32(i)
		msg.WithRequestTimeout(time.Millisecond * time.Duration(i))
	}
	if v := head.Request.Header.Get(TrpcCaller); v != "" {
		trpcReq.Caller = []byte(v)
		msg.WithCallerServiceName(v)
	}
	if v := head.Request.Header.Get(TrpcCallee); v != "" {
		trpcReq.Callee = []byte(v)
		msg.WithCalleeServiceName(v)
	}

	msg.WithDyeing((trpcReq.GetMessageType() & uint32(trpc.TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

	if v := head.Request.Header.Get(TrpcTransInfo); v != "" {
		return setTransInfo(trpcReq, msg, v)
	}
	return nil
}

func setTransInfo(trpcReq *trpc.RequestProtocol, msg codec.Msg, v string) error {
	m := make(map[string]string)
	if err := codec.Unmarshal(codec.SerializationTypeJSON, []byte(v), &m); err != nil {
		return err
	}
	trpcReq.TransInfo = make(map[string][]byte)
	// 由于http header只能传明文字符串，但是trpc transinfo是二进制流，所以需要经过base64保护一下
	for k, v := range m {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			decoded = []byte(v)
		}
		trpcReq.TransInfo[k] = decoded

		if k == TrpcEnv {
			msg.WithEnvTransfer(string(decoded))
		}
		if k == TrpcDyeingKey {
			msg.WithDyeingKey(string(decoded))
		}
	}
	msg.WithServerMetaData(trpcReq.GetTransInfo())
	return nil
}

// getReqbody 获取请求 body
func (sc *ServerCodec) getReqbody(head *Header, msg codec.Msg) ([]byte, error) {
	msg.WithCalleeMethod(head.Request.URL.Path)
	msg.WithServerRPCName(head.Request.URL.Path)

	if !sc.AutoReadBody {
		return nil, nil
	}

	var reqbody []byte
	if head.Request.Method == stdhttp.MethodGet {
		msg.WithSerializationType(codec.SerializationTypeGet)
		reqbody = []byte(head.Request.URL.RawQuery)
	} else {
		var exist bool
		msg.WithSerializationType(codec.SerializationTypeJSON)
		ct := head.Request.Header.Get("Content-Type")
		for contentType, serializationType := range contentTypeSerializationType {
			if strings.Contains(ct, contentType) {
				msg.WithSerializationType(serializationType)
				exist = true
				break
			}
		}
		if exist {
			var err error
			reqbody, err = getBody(ct, head.Request)
			if err != nil {
				return nil, err
			}
		}
	}
	head.ReqBody = reqbody
	return reqbody, nil
}

// getBody 获取请求 body
func getBody(contentType string, r *stdhttp.Request) ([]byte, error) {
	if strings.Contains(contentType, serializationTypeContentType[codec.SerializationTypeFormData]) {
		if r.Form == nil {
			if err := r.ParseMultipartForm(ParseMultipartFormMaxMemory); err != nil {
				return nil, fmt.Errorf("parse multipart form: %w", err)
			}
		}
		return []byte(r.Form.Encode()), nil
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("body readAll: %w", err)
	}
	return body, nil
}

// updateMsg 更新 msg
func (sc *ServerCodec) updateMsg(head *Header, msg codec.Msg) {
	ce := head.Request.Header.Get("Content-Encoding")
	if ce != "" {
		msg.WithCompressType(contentEncodingCompressType[ce])
	}

	// 上游
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName("trpc.http.upserver.upservice")
	}

	// 自身
	if msg.CalleeServiceName() == "" {
		msg.WithCalleeServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}
}

// Decode 解码http包头 http server transport 已经把request所有数据塞到ctx里面了， 这里的reqbuf是空的
func (sc *ServerCodec) Decode(msg codec.Msg, _ []byte) ([]byte, error) {
	head := Head(msg.Context())
	if head == nil {
		return nil, errors.New("server decode missing http header in context")
	}

	reqbody, err := sc.getReqbody(head, msg)
	if err != nil {
		return nil, err
	}
	if err := sc.setReqHeader(head, msg); err != nil {
		return nil, err
	}

	sc.updateMsg(head, msg)
	return reqbody, nil
}

// ErrorHandler http server回包错误处理函数，默认将错误码放在header里面，可以自己替换成具体实现
type ErrorHandler func(w stdhttp.ResponseWriter, r *stdhttp.Request, e *errs.Error)

var defaultErrHandler = func(w stdhttp.ResponseWriter, r *stdhttp.Request, e *errs.Error) {
	errMsg := strings.Replace(e.Msg, "\r", "\\r", -1)
	errMsg = strings.Replace(errMsg, "\n", "\\n", -1)

	w.Header().Add("trpc-error-msg", errMsg)
	if e.Type == errs.ErrorTypeFramework {
		w.Header().Add("trpc-ret", strconv.Itoa(int(e.Code)))
	} else {
		w.Header().Add("trpc-func-ret", strconv.Itoa(int(e.Code)))
	}

	if code, ok := ErrsToHTTPStatus[e.Code]; ok {
		w.WriteHeader(code)
	}
}

// ResponseHandler http server 回包数据处理函数，默认将内容直接返回，可以自己替换成具体实现
type ResponseHandler func(w stdhttp.ResponseWriter, r *stdhttp.Request, rspbody []byte) error

var defaultRspHandler = func(w stdhttp.ResponseWriter, r *stdhttp.Request, rspbody []byte) error {
	if len(rspbody) == 0 {
		return nil
	}
	if _, err := w.Write(rspbody); err != nil {
		return fmt.Errorf("http write response error: %s", err.Error())
	}
	return nil
}

// Encode 设置http包头 回包buffer已经写到header里面的response writer，不需要返回rspbuf
func (sc *ServerCodec) Encode(msg codec.Msg, rspbody []byte) (b []byte, err error) {
	head := Head(msg.Context())
	if head == nil {
		return nil, ErrEncodeMissingHeader
	}
	req := head.Request
	rsp := head.Response
	ctKey := "Content-Type"

	rsp.Header().Add("X-Content-Type-Options", "nosniff")
	ct := rsp.Header().Get(ctKey)
	if ct == "" {
		ct = req.Header.Get(ctKey)
		if req.Method == stdhttp.MethodGet || ct == "" {
			ct = "application/json"
		}
		rsp.Header().Add(ctKey, ct)
	}
	if strings.Contains(ct, serializationTypeContentType[codec.SerializationTypeFormData]) {
		formDataCt := getFormDataContentType()
		rsp.Header().Set(ctKey, formDataCt)
	}

	if len(msg.ServerMetaData()) > 0 {
		m := make(map[string]string)
		for k, v := range msg.ServerMetaData() {
			m[k] = base64.StdEncoding.EncodeToString(v)
		}
		val, _ := codec.Marshal(codec.SerializationTypeJSON, m)
		rsp.Header().Set("trpc-trans-info", string(val))
	}

	if msg.CompressType() > 0 { // 回包告诉客户端使用什么解压缩方式
		rsp.Header().Add("Content-Encoding", compressTypeContentEncoding[msg.CompressType()])
	}

	// 1. 先处理异常情况，只要 server 返回 error 就不再处理返回的数据
	if e := msg.ServerRspErr(); e != nil {
		if sc.ErrHandler != nil {
			sc.ErrHandler(rsp, req, e)
		}
		return
	}
	// 2. 处理正常情况下的数据返回
	if sc.RspHandler != nil {
		if err := sc.RspHandler(rsp, req, rspbody); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ClientCodec 解码http client请求
type ClientCodec struct{}

// Encode 设置http client请求的元数据 client 已经序列化并压缩好传入reqbody
func (c *ClientCodec) Encode(msg codec.Msg, reqbody []byte) ([]byte, error) {
	var reqHeader *ClientReqHeader
	if msg.ClientReqHead() != nil { // 用户自己设置了http client req header
		httpReqHeader, ok := msg.ClientReqHead().(*ClientReqHeader)
		if !ok {
			return nil, errors.New("http header must be type of *http.ClientReqHeader")
		}
		reqHeader = httpReqHeader
	} else {
		reqHeader = &ClientReqHeader{}
		msg.WithClientReqHead(reqHeader)
	}

	if reqHeader.Method == "" {
		if len(reqbody) == 0 {
			reqHeader.Method = stdhttp.MethodGet
		} else {
			reqHeader.Method = stdhttp.MethodPost
		}
	}

	if msg.ClientRspHead() != nil { // 用户自己设置了http client rsp header
		_, ok := msg.ClientRspHead().(*ClientRspHeader)
		if !ok {
			return nil, errors.New("http header must be type of *http.ClientRspHeader")
		}
	} else {
		msg.WithClientRspHead(&ClientRspHeader{})
	}

	c.updateMsg(msg)
	return reqbody, nil
}

// Decode 解析http client回包里的元数据
func (c *ClientCodec) Decode(msg codec.Msg, _ []byte) (rspBody []byte, err error) {
	rspHeader, ok := msg.ClientRspHead().(*ClientRspHeader)
	if !ok {
		return nil, errors.New("rsp header must be type of *http.ClientRspHeader")
	}

	rsp := rspHeader.Response
	if rsp.Body != nil {
		defer rsp.Body.Close()
	}
	if val := rsp.Header.Get("Content-Encoding"); val != "" {
		msg.WithCompressType(contentEncodingCompressType[val])
	}
	ct := rsp.Header.Get("Content-Type")
	for contentType, serializationType := range contentTypeSerializationType {
		if strings.Contains(ct, contentType) {
			msg.WithSerializationType(serializationType)
			break
		}
	}
	if val := rsp.Header.Get("trpc-ret"); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			e := &errs.Error{
				Type: errs.ErrorTypeCalleeFramework,
				Code: int32(i),
				Desc: "trpc",
				Msg:  rsp.Header.Get("trpc-error-msg"),
			}
			msg.WithClientRspErr(e)
			return nil, nil
		}
	}
	if val := rsp.Header.Get("trpc-func-ret"); val != "" {
		i, _ := strconv.Atoi(val)
		if i != 0 {
			msg.WithClientRspErr(errs.New(i, rsp.Header.Get("trpc-error-msg")))
			return nil, nil
		}
	}
	if rsp.StatusCode >= stdhttp.StatusMultipleChoices {
		e := &errs.Error{
			Type: errs.ErrorTypeBusiness,
			Code: int32(rsp.StatusCode),
			Desc: "http",
			Msg:  "http client codec StatusCode: " + stdhttp.StatusText(rsp.StatusCode),
		}
		msg.WithClientRspErr(e)
		return nil, nil
	}

	body, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("readall http body fail:%s", err.Error())
	}
	return body, nil
}

// updateMsg 更新消息 msg
func (c *ClientCodec) updateMsg(msg codec.Msg) {
	// 自身
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.http.%s.service", path.Base(os.Args[0])))
	}
}
