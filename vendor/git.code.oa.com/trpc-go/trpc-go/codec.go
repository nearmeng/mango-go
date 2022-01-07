package trpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sync/atomic"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/transport"

	"github.com/golang/protobuf/proto"
)

func init() {
	codec.Register("trpc", DefaultServerCodec, DefaultClientCodec)
	transport.RegisterFramerBuilder("trpc", DefaultFramerBuilder)
}

// default codec
var (
	DefaultServerCodec   = &ServerCodec{streamCodec: NewServerStreamCodec()}
	DefaultClientCodec   = &ClientCodec{streamCodec: NewClientStreamCodec()}
	DefaultFramerBuilder = &FramerBuilder{}
)

// DefaultMaxFrameSize 默认帧最大10M
var DefaultMaxFrameSize = 10 * 1024 * 1024

// trpc protocol codec
// 具体协议格式参考：https://git.code.oa.com/trpc/trpc-protocol/blob/master/docs/protocol_design.md
const (
	// 起始魔数
	stx = uint16(TrpcMagic_TRPC_MAGIC_VALUE)
	// 帧头 2 bytes stx + 1 byte type + 1 byte stream frame type + 4 bytes total len
	// + 2 bytes pb header len + 4 bytes stream id + 2 bytes reserved
	frameHeadLen = uint16(16)

	DyeingKey   = "trpc-dyeing-key" // 染色key
	UserIP      = "trpc-user-ip"    // 客户端最上游ip
	EnvTransfer = "trpc-env"        // 透传环境数据
)

// FrameHead trpc 帧头信息（未包含起始魔数0x930)
type FrameHead struct {
	FrameType       uint8
	StreamFrameType uint8
	StreamID        uint32
	FrameReserved   uint16
}

// FramerBuilder 数据帧构造器
type FramerBuilder struct{}

// New 生成一个trpc数据帧
func (fb *FramerBuilder) New(reader io.Reader) codec.Framer {
	return &framer{
		reader: reader,
	}
}

// framer trpc帧读取
type framer struct {
	reader io.Reader
	header [frameHeadLen]byte
}

// ReadFrame 从io reader拆分出完整数据桢
func (f *framer) ReadFrame() (msgbuf []byte, err error) {
	num, err := io.ReadFull(f.reader, f.header[:])
	if err != nil {
		return nil, err
	}
	if num != int(frameHeadLen) {
		return nil, fmt.Errorf("trpc framer: read frame header num %d != %d, invalid", num, int(frameHeadLen))
	}
	magic := binary.BigEndian.Uint16(f.header[:2])
	if magic != uint16(TrpcMagic_TRPC_MAGIC_VALUE) {
		return nil, fmt.Errorf(
			"trpc framer: read framer head magic %d != %d, not match", magic, uint16(TrpcMagic_TRPC_MAGIC_VALUE))
	}
	totalLen := binary.BigEndian.Uint32(f.header[4:8])
	if totalLen < uint32(frameHeadLen) {
		return nil, fmt.Errorf(
			"trpc framer: read frame header total len %d < %d, invalid", totalLen, uint32(frameHeadLen))
	}

	if totalLen > uint32(DefaultMaxFrameSize) {
		return nil, fmt.Errorf(
			"trpc framer: read frame header total len %d > %d, too large", totalLen, uint32(DefaultMaxFrameSize))
	}

	msg := make([]byte, totalLen)
	num, err = io.ReadFull(f.reader, msg[frameHeadLen:totalLen])
	if err != nil {
		return nil, err
	}
	if num != int(totalLen-uint32(frameHeadLen)) {
		return nil, fmt.Errorf(
			"trpc framer: read frame total num %d != %d, invalid", num, int(totalLen-uint32(frameHeadLen)))
	}
	copy(msg, f.header[:])
	return msg, nil
}

// IsSafe Framer支持并发安全读包
func (f *framer) IsSafe() bool {
	return true
}

// ServerCodec trpc服务端编解码
type ServerCodec struct {
	streamCodec *ServerStreamCodec
}

// Decode 服务端收到客户端二进制请求数据解包到reqbody, service handler会自动创建一个新的空的msg 作为初始通用消息体
func (s *ServerCodec) Decode(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	if len(reqbuf) < int(frameHeadLen) {
		return nil, errors.New("server decode req buf len invalid")
	}
	frameHead := &FrameHead{
		FrameType:       reqbuf[2],
		StreamFrameType: reqbuf[3],
		StreamID:        binary.BigEndian.Uint32(reqbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(reqbuf[14:16]),
	}
	msg.WithFrameHead(frameHead)
	if frameHead.FrameType != uint8(TrpcDataFrameType_TRPC_UNARY_FRAME) {
		return s.streamCodec.Decode(msg, reqbuf)
	}

	pbHeadLen := binary.BigEndian.Uint16(reqbuf[8:10])
	if pbHeadLen == 0 {
		return nil, errors.New("server decode pb head len empty")
	}
	begin := int(frameHeadLen)
	end := int(frameHeadLen) + int(pbHeadLen)
	if end > len(reqbuf) {
		return nil, errors.New("server decode pb head len invalid")
	}

	req := &RequestProtocol{}
	if err := proto.Unmarshal(reqbuf[begin:end], req); err != nil {
		return nil, err
	}
	// 解请求包体
	reqbody := reqbuf[end:]

	// 提前构造响应包头
	rsp := &ResponseProtocol{
		Version:         uint32(TrpcProtoVersion_TRPC_PROTO_V1),
		CallType:        req.CallType,
		RequestId:       req.RequestId,
		MessageType:     req.MessageType,
		ContentType:     req.ContentType,
		ContentEncoding: req.ContentEncoding,
	}

	s.updateMsg(msg, req, rsp)
	return reqbody, nil
}

// Encode 服务端打包rspbody到二进制 回给客户端
func (s *ServerCodec) Encode(msg codec.Msg, rspbody []byte) (rspbuf []byte, err error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if ok && frameHead != nil && frameHead.FrameType != uint8(TrpcDataFrameType_TRPC_UNARY_FRAME) {
		return s.streamCodec.Encode(msg, rspbody)
	}
	// 取出回包包头
	rsp, ok := msg.ServerRspHead().(*ResponseProtocol)
	if !ok {
		rsp = &ResponseProtocol{}
	}

	// 更新序列化类型和压缩类型
	rsp.ContentType = uint32(msg.SerializationType())
	rsp.ContentEncoding = uint32(msg.CompressType())

	// 将处理函数handler返回的error转成协议包头里面的错误码字段
	if e := msg.ServerRspErr(); e != nil {
		rsp.ErrorMsg = []byte(e.Msg)
		if e.Type == errs.ErrorTypeFramework {
			rsp.Ret = e.Code
		} else {
			rsp.FuncRet = e.Code
		}
	}

	if len(msg.ServerMetaData()) > 0 {
		if rsp.TransInfo == nil {
			rsp.TransInfo = make(map[string][]byte)
		}
		for k, v := range msg.ServerMetaData() {
			rsp.TransInfo[k] = v
		}
	}

	rsphead, err := proto.Marshal(rsp)
	if err != nil {
		return nil, err
	}

	if rspbuf, err = writeFrame(msg, rsphead, rspbody); err == errHeadOverflowsUint16 {
		// 放弃所有 TransInfo，并返回 RetServerEncodeFail。
		// 无论原始请求成功与否，我们都将覆盖原始错误。
		rsp.TransInfo = nil
		rsp.Ret = errs.RetServerEncodeFail
		rsp.ErrorMsg = []byte(err.Error())
		if rsphead, err = proto.Marshal(rsp); err != nil {
			return nil, err
		}
		// 如果还失败，就只能放弃回包了。最外层会通过关闭连接通知 client。
		return writeFrame(msg, rsphead, rspbody)
	}
	return rspbuf, err
}

func (s *ServerCodec) updateMsg(msg codec.Msg, req *RequestProtocol, rsp *ResponseProtocol) {
	// 设置具体业务协议请求包头
	msg.WithServerReqHead(req)
	msg.WithServerRspHead(rsp)

	//-----------------以下为trpc框架需要的数据-----------------------------//
	// 设置上游允许的超时时间
	msg.WithRequestTimeout(time.Millisecond * time.Duration(req.GetTimeout()))
	// 设置上游服务名
	msg.WithCallerServiceName(string(req.GetCaller()))
	msg.WithCalleeServiceName(string(req.GetCallee()))
	// 设置当前请求的rpc方法名(命令字)
	msg.WithServerRPCName(string(req.GetFunc()))
	// 设置body的序列化方式
	msg.WithSerializationType(int(req.GetContentType()))
	// 设置body压缩方式
	msg.WithCompressType(int(req.GetContentEncoding()))
	// 设置染色标记
	msg.WithDyeing((req.GetMessageType() & uint32(TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)
	// 解析tracing MetaData，设置MetaData到msg
	if len(req.TransInfo) > 0 {
		msg.WithServerMetaData(req.GetTransInfo())
		// 染色标记
		if bs, ok := req.TransInfo[DyeingKey]; ok {
			msg.WithDyeingKey(string(bs))
		}
		// 透传环境信息
		if envs, ok := req.TransInfo[EnvTransfer]; ok {
			msg.WithEnvTransfer(string(envs))
		}
	}
	// 设置请求类型
	msg.WithCallType(codec.RequestType(req.GetCallType()))
}

// ClientCodec trpc客户端编解码
type ClientCodec struct {
	streamCodec *ClientStreamCodec
	RequestID   uint32 //全局唯一request id
}

// updateReqHead 更新请求头
func (c *ClientCodec) updateReqHead(msg codec.Msg, req *RequestProtocol) {
	// 设置调用方 service name
	req.Caller = []byte(msg.CallerServiceName())
	// 设置被调方 service name
	req.Callee = []byte(msg.CalleeServiceName())
	// 设置后端函数rpc方法名，由client stub外层设置
	req.Func = []byte(msg.ClientRPCName())
	// 设置后端序列化方式
	req.ContentType = uint32(msg.SerializationType())
	// 设置后端解压缩方式
	req.ContentEncoding = uint32(msg.CompressType())
	// 设置下游剩余超时时间
	req.Timeout = uint32(msg.RequestTimeout() / time.Millisecond)
	// 设置染色信息
	if msg.Dyeing() {
		req.MessageType = req.MessageType | uint32(TrpcMessageType_TRPC_DYEING_MESSAGE)
	}
	// 设置client的transinfo
	req.TransInfo = setClientTransInfo(msg, req.TransInfo)
	// 设置请求类型
	req.CallType = uint32(msg.CallType())
}

// setClientTransInfo 设置Client请求的Transinfo信息
func setClientTransInfo(msg codec.Msg, trans map[string][]byte) map[string][]byte {
	// 设置MetaData
	if len(msg.ClientMetaData()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		for k, v := range msg.ClientMetaData() {
			trans[k] = v
		}
	}
	if len(msg.DyeingKey()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		trans[DyeingKey] = []byte(msg.DyeingKey())
	}
	if len(msg.EnvTransfer()) > 0 {
		if trans == nil {
			trans = make(map[string][]byte)
		}
		trans[EnvTransfer] = []byte(msg.EnvTransfer())
	} else {
		// 如果msg.EnvTransfer()为空，需要清空req.TransInfo的透传环境信息
		if _, ok := trans[EnvTransfer]; ok {
			trans[EnvTransfer] = nil
		}
	}
	return trans
}

// Encode 客户端打包reqbody到二进制数据 发到服务端, client stub会自动clone生成新的msg
func (c *ClientCodec) Encode(msg codec.Msg, reqbody []byte) (reqbuf []byte, err error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if ok && frameHead != nil && frameHead.FrameType != uint8(TrpcDataFrameType_TRPC_UNARY_FRAME) {
		return c.streamCodec.Encode(msg, reqbody)
	}

	req, err := c.getRequestHead(msg)
	if err != nil {
		return nil, err
	}

	// 框架自动生成全局唯一request id
	req.RequestId = atomic.AddUint32(&c.RequestID, 1)

	c.updateMsg(msg, req)
	c.updateReqHead(msg, req)

	reqhead, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return writeFrame(msg, reqhead, reqbody)
}

func (c *ClientCodec) getRequestHead(msg codec.Msg) (*RequestProtocol, error) {
	// 构造后端请求包头
	if msg.ClientReqHead() != nil {
		// client req head不为空 说明是用户自己创建，直接使用即可
		req, ok := msg.ClientReqHead().(*RequestProtocol)
		if !ok {
			return nil, errors.New("client encode req head type invalid")
		}
		return req, nil
	}

	req := &RequestProtocol{
		Version:  uint32(TrpcProtoVersion_TRPC_PROTO_V1),
		CallType: uint32(TrpcCallType_TRPC_UNARY_CALL),
	}
	// 如果serverReqHead有数据,需要复制MessageType字段和TransInfo字段
	if serverReq, ok := msg.ServerReqHead().(*RequestProtocol); ok {
		if len(serverReq.TransInfo) > 0 {
			req.TransInfo = make(map[string][]byte)
		}
		for k, v := range serverReq.TransInfo {
			req.TransInfo[k] = v
		}
		req.MessageType = serverReq.MessageType
		return req, nil
	}
	// 保存新的client req head
	msg.WithClientReqHead(req)
	return req, nil
}

var (
	errHeadOverflowsUint16 = errors.New("head len overflows uint16")
	errFrameTooLarge       = fmt.Errorf("frame len is larger than %d", DefaultMaxFrameSize)
)

// writeFrame将msg和head，body根据字节序序列化到相应的帧里面
func writeFrame(msg codec.Msg, head, body []byte) ([]byte, error) {
	if len(head) > math.MaxUint16 {
		return nil, errHeadOverflowsUint16
	}
	headLen := uint16(len(head))
	if int64(frameHeadLen)+int64(headLen)+int64(len(body)) > int64(DefaultMaxFrameSize) {
		return nil, errFrameTooLarge
	}
	totalLen := uint32(frameHeadLen) + uint32(headLen) + uint32(len(body))

	// 创建相应的buffer
	buf := make([]byte, totalLen)
	// magic
	binary.BigEndian.PutUint16(buf[:2], stx)

	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok {
		frameHead = &FrameHead{}
	}
	// frameType
	buf[2] = frameHead.FrameType
	// StreamFrameType
	buf[3] = frameHead.StreamFrameType
	// totalen
	binary.BigEndian.PutUint32(buf[4:8], totalLen)
	// pb head length
	binary.BigEndian.PutUint16(buf[8:10], headLen)
	// StreamID
	binary.BigEndian.PutUint32(buf[10:14], frameHead.StreamID)
	// 包头无字节序问题，直接拷贝
	copy(buf[16:16+headLen], head)
	// 包体无字节序问题，直接拷贝
	copy(buf[16+headLen:], body)
	return buf, nil
}

func (c *ClientCodec) updateMsg(msg codec.Msg, req *RequestProtocol) {
	// 如果调用方为空 则取进程名, client小工具，没有caller
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.app.%s.service", path.Base(os.Args[0])))
	}

	// 设置 RequestID
	msg.WithRequestID(req.RequestId)
}

// Decode 客户端收到服务端二进制回包数据解包到rspbody
func (c *ClientCodec) Decode(msg codec.Msg, rspbuf []byte) (rspbody []byte, err error) {
	if len(rspbuf) < int(frameHeadLen) {
		return nil, errors.New("client decode rsp buf len invalid")
	}
	frameHead := &FrameHead{
		FrameType:       rspbuf[2],
		StreamFrameType: rspbuf[3],
		StreamID:        binary.BigEndian.Uint32(rspbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(rspbuf[14:16]),
	}
	if frameHead.FrameType != uint8(TrpcDataFrameType_TRPC_UNARY_FRAME) {
		return c.streamCodec.Decode(msg, rspbuf)
	}

	// 构造后端响应包头
	var rsp *ResponseProtocol
	if msg.ClientRspHead() != nil {
		// client rsp head不为空 说明是用户故意创建，希望底层回传后端响应包头
		response, ok := msg.ClientRspHead().(*ResponseProtocol)
		if !ok {
			return nil, errors.New("client decode rsp head type invalid")
		}
		rsp = response
	} else {
		// client rsp head为空 说明用户不关心后端响应包头
		rsp = &ResponseProtocol{}
		// 保存新的client rsp head
		msg.WithClientRspHead(rsp)
	}

	// 解响应包头
	pbHeadLen := binary.BigEndian.Uint16(rspbuf[8:10])
	if pbHeadLen == 0 {
		return nil, errors.New("client decode pb head len empty")
	}
	begin := int(frameHeadLen)
	end := int(frameHeadLen) + int(pbHeadLen)
	if end > len(rspbuf) {
		return nil, errors.New("server decode pb head len invalid")
	}
	if err := proto.Unmarshal(rspbuf[begin:end], rsp); err != nil {
		return nil, err
	}

	frameHead = &FrameHead{
		FrameType:       rspbuf[2],
		StreamFrameType: rspbuf[3],
		StreamID:        binary.BigEndian.Uint32(rspbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(rspbuf[14:16]),
	}
	if err := updateMsg(frameHead, rsp, msg); err != nil {
		return nil, err
	}

	// 解响应包体
	rspbody = rspbuf[end:]
	return rspbody, nil
}

// FrameResponse Decode 的结构体
type FrameResponse struct {
	frameHead  *FrameHead
	packetHead *ResponseProtocol
	packetBody []byte
}

// GetRequestID 返回 request id
func (rsp *FrameResponse) GetRequestID() uint32 {
	return rsp.packetHead.GetRequestId()
}

// GetResponseBuf 返回包体
func (rsp *FrameResponse) GetResponseBuf() []byte {
	return rsp.packetBody
}

// Decode 从io reader拆分出完整数据桢
func (f *framer) Decode() (codec.TransportResponseFrame, error) {
	rspbuf, err := f.ReadFrame()
	if err != nil {
		return nil, err
	}

	frameHead := &FrameHead{
		FrameType:       rspbuf[2],
		StreamFrameType: rspbuf[3],
		StreamID:        binary.BigEndian.Uint32(rspbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(rspbuf[14:16]),
	}

	packetHead := &ResponseProtocol{}
	// 解响应包头
	pbHeadLen := binary.BigEndian.Uint16(rspbuf[8:10])
	if pbHeadLen == 0 {
		return nil, errors.New("client decode pb head len empty")
	}
	begin := int(frameHeadLen)
	end := int(frameHeadLen) + int(pbHeadLen)
	if end > len(rspbuf) {
		return nil, errors.New("client decode pb head len invalid")
	}
	if err := proto.Unmarshal(rspbuf[begin:end], packetHead); err != nil {
		return nil, err
	}

	rspbody := rspbuf[end:]
	return &FrameResponse{
		frameHead:  frameHead,
		packetHead: packetHead,
		packetBody: rspbody,
	}, nil
}

func copyRspHead(dst, src *ResponseProtocol) {
	dst.Version = src.Version
	dst.CallType = src.CallType
	dst.RequestId = src.RequestId
	dst.Ret = src.Ret
	dst.FuncRet = src.FuncRet
	dst.ErrorMsg = src.ErrorMsg
	dst.MessageType = src.MessageType
	dst.TransInfo = src.TransInfo
	dst.ContentType = src.ContentType
	dst.ContentEncoding = src.ContentEncoding
}

func updateMsg(frameHead *FrameHead, rsp *ResponseProtocol, msg codec.Msg) error {
	msg.WithFrameHead(frameHead)
	msg.WithCompressType(int(rsp.GetContentEncoding()))
	msg.WithSerializationType(int(rsp.GetContentType()))

	if len(rsp.TransInfo) > 0 { // 重新设置透传字段，下游有可能返回新的透传信息
		md := msg.ClientMetaData()
		if len(md) == 0 {
			md = codec.MetaData{}
		}
		for k, v := range rsp.TransInfo {
			md[k] = v
		}
		msg.WithClientMetaData(md)
	}

	// 将业务协议包头错误码转化成err返回给调用用户
	if rsp.GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: rsp.GetRet(),
			Desc: "trpc",
			Msg:  string(rsp.GetErrorMsg()),
		}
		msg.WithClientRspErr(e)
	} else if rsp.GetFuncRet() != 0 {
		msg.WithClientRspErr(errs.New(int(rsp.GetFuncRet()), string(rsp.GetErrorMsg())))
	}

	if req, ok := msg.ClientReqHead().(*RequestProtocol); ok {
		if req.GetRequestId() != rsp.GetRequestId() {
			return errors.New("rsp request_id different from req request_id")
		}
	}
	return nil
}

// UpdateMsg 更新 msg
func (f *framer) UpdateMsg(res interface{}, msg codec.Msg) error {
	r, ok := res.(*FrameResponse)
	if !ok {
		return errors.New("update msg invalid rsp type")
	}

	// 构造后端响应包头
	var rsp *ResponseProtocol
	if msg.ClientRspHead() != nil {
		// client rsp head不为空 说明是用户故意创建，希望底层回传后端响应包头
		response, ok := msg.ClientRspHead().(*ResponseProtocol)
		if !ok {
			return errors.New("client decode rsp head type invalid")
		}
		rsp = response
		copyRspHead(rsp, r.packetHead)
	} else {
		// client rsp head为空 说明用户不关心后端响应包头
		rsp = r.packetHead
		// 保存新的client rsp head
		msg.WithClientRspHead(rsp)
	}
	return updateMsg(r.frameHead, rsp, msg)
}
