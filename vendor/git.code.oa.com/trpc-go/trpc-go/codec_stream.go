package trpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"

	"github.com/golang/protobuf/proto"
)

var (
	// 找不到对应的帧的类型
	errUnknownFrameType error = errors.New("unkonwn stream frame type")
	// 客户端decode帧长度不合法
	errClientDecodeTotalLength error = errors.New("client decode total length invalid")
	// Close帧encode出错
	errEncodeCloseFrame error = errors.New("encode close frame error")
	// feedback帧encode出错
	errEncodeFeedbackFrame error = errors.New("encode feedback error")
	// 当前帧找不到对应的init元数据信息
	errUninitializedMeta error = errors.New("uninitialized meta")
	// frameHead的类型断言错误，不是trpc的frameHead
	errFrameHeadTypeInvalid error = errors.New("framehead type invalid")
)

// NewServerStreamCodec 新建流式服务端编解码器
func NewServerStreamCodec() *ServerStreamCodec {
	return &ServerStreamCodec{initMetas: make(map[net.Addr]map[uint32]*TrpcStreamInitMeta), m: &sync.RWMutex{}}
}

// NewClientStreamCodec 新建流式客户端编解码器
func NewClientStreamCodec() *ClientStreamCodec {
	return &ClientStreamCodec{}
}

// ServerStreamCodec trpc服务端流式编解码
type ServerStreamCodec struct {
	m         *sync.RWMutex
	initMetas map[net.Addr]map[uint32]*TrpcStreamInitMeta //initMetas addr->streamID->保存initMeta的映射
}

// ClientStreamCodec 客户端编码器的实现
type ClientStreamCodec struct {
}

// Encode 流式客户端编码入口
func (c *ClientStreamCodec) Encode(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	frameHead := getFrameHead(msg)
	switch TrpcStreamFrameType(frameHead.StreamFrameType) {
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		// Init帧
		return c.encodeInitFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		// 数据包
		return c.encodeDataFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		// Close帧
		return c.encodeCloseFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		// feedback帧
		return c.encodeFeedbackFrame(frameHead, msg, reqbuf)
	default:
		return nil, errUnknownFrameType
	}
}

// Decode 流式客户端解码入口
func (c *ClientStreamCodec) Decode(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	totalLength := binary.BigEndian.Uint32(rspbuf[4:8])
	if totalLength < uint32(frameHeadLen) {
		return nil, errClientDecodeTotalLength
	}
	// 将帧头解出
	frameHead := &FrameHead{
		FrameType:       rspbuf[2],
		StreamFrameType: rspbuf[3],
		StreamID:        binary.BigEndian.Uint32(rspbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(rspbuf[14:16]),
	}
	msg.WithFrameHead(frameHead)
	msg.WithStreamID(frameHead.StreamID)

	//根据不同数据类型进行分发
	switch TrpcStreamFrameType(frameHead.StreamFrameType) {
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return c.decodeInitFrame(msg, rspbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return c.decodeDataFrame(msg, rspbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return c.decodeCloseFrame(msg, rspbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return c.decodeFeedbackFrame(msg, rspbuf)
	default:
		return nil, errUnknownFrameType
	}
}

// decodeCloseFrame 解码Close帧
func (c *ClientStreamCodec) decodeCloseFrame(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	// 将Close帧反序列化出来
	close := &TrpcStreamCloseMeta{}
	if err := proto.Unmarshal(rspbuf[frameHeadLen:], close); err != nil {
		return nil, err
	}

	// 判断是否为Reset的Close帧，或者返回码不为0，则认为是个异常的关闭，将error返回给客户端
	if close.GetCloseType() == int32(TrpcStreamCloseType_TRPC_STREAM_RESET) || close.GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: close.GetRet(),
			Desc: "trpc",
			Msg:  string(close.GetMsg()),
		}
		msg.WithClientRspErr(e)
	}
	msg.WithStreamFrame(close)
	return nil, nil
}

// decodeFeedbackFrame 解出feedback帧
func (c *ClientStreamCodec) decodeFeedbackFrame(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	feedback := &TrpcStreamFeedBackMeta{}
	if err := proto.Unmarshal(rspbuf[frameHeadLen:], feedback); err != nil {
		return nil, err
	}
	msg.WithStreamFrame(feedback)
	return nil, nil
}

// decodeInitFrame 流式客户端解码Init帧
func (c *ClientStreamCodec) decodeInitFrame(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	// Init帧的数据结构，trpc.pb.go 里面的定义
	initMeta := &TrpcStreamInitMeta{}
	if err := proto.Unmarshal(rspbuf[frameHeadLen:], initMeta); err != nil {
		return nil, err
	}

	msg.WithCompressType(int(initMeta.GetContentEncoding()))
	msg.WithSerializationType(int(initMeta.GetContentType()))

	// 如果流式的服务端返回错误
	if initMeta.GetResponseMeta().GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: initMeta.GetResponseMeta().GetRet(),
			Desc: "trpc",
			Msg:  string(initMeta.GetResponseMeta().GetErrorMsg()),
		}
		msg.WithClientRspErr(e)
	}
	msg.WithStreamFrame(initMeta)
	return nil, nil

}

// decodeDataFrame 解出数据帧
func (c *ClientStreamCodec) decodeDataFrame(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	// 数据帧比较简单，直接返回帧头之后的数据帧
	return rspbuf[frameHeadLen:], nil
}

// encodeInitFrame 流式编码init帧
func (c *ClientStreamCodec) encodeInitFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	initMeta, ok := msg.StreamFrame().(*TrpcStreamInitMeta)
	if !ok {
		initMeta = &TrpcStreamInitMeta{}
		initMeta.RequestMeta = &TrpcStreamInitRequestMeta{}
	}
	req := initMeta.RequestMeta
	// 如果调用方为空 则取进程名, client小工具，没有caller
	if msg.CallerServiceName() == "" {
		msg.WithCallerServiceName(fmt.Sprintf("trpc.app.%s.service", path.Base(os.Args[0])))
	}
	req.Caller = []byte(msg.CallerServiceName())
	// 设置被调方 service name
	req.Callee = []byte(msg.CalleeServiceName())
	// 设置后端函数rpc方法名，由client stub外层设置
	req.Func = []byte(msg.ClientRPCName())
	// 设置后端序列化方式
	initMeta.ContentType = uint32(msg.SerializationType())
	// 设置后端解压缩方式
	initMeta.ContentEncoding = uint32(msg.CompressType())
	// 设置染色信息
	if msg.Dyeing() {
		req.MessageType = req.MessageType | uint32(TrpcMessageType_TRPC_DYEING_MESSAGE)
	}
	// 设置client的transinfo
	req.TransInfo = setClientTransInfo(msg, req.TransInfo)
	streamBuf, err := proto.Marshal(initMeta)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeDataFrame  客户端流式编码数据帧
func (c *ClientStreamCodec) encodeDataFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	return frameWrite(frameHead, reqbuf)
}

// encodeCloseFrame 客户端流式编码Close帧
func (c *ClientStreamCodec) encodeCloseFrame(frameHead *FrameHead, msg codec.Msg,
	reqbuf []byte) (rspbuf []byte, err error) {
	closeFrame, ok := msg.StreamFrame().(*TrpcStreamCloseMeta)
	if !ok {
		return nil, errEncodeCloseFrame
	}
	streamBuf, err := proto.Marshal(closeFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeFeedbackFrame 客户端编码feedback帧
func (c *ClientStreamCodec) encodeFeedbackFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	feedbackFrame, ok := msg.StreamFrame().(*TrpcStreamFeedBackMeta)
	if !ok {
		return nil, errEncodeFeedbackFrame
	}
	streamBuf, err := proto.Marshal(feedbackFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// frameWrite 将帧头转成二进制帧
func frameWrite(frameHead *FrameHead, streamBuf []byte) ([]byte, error) {
	streamLen := len(streamBuf)
	totalLen := uint32(streamLen + int(frameHeadLen))
	buf := make([]byte, totalLen)
	// magic
	binary.BigEndian.PutUint16(buf[:2], stx)
	// frameType
	buf[2] = frameHead.FrameType
	// StreamFrameType
	buf[3] = frameHead.StreamFrameType
	// totalen
	binary.BigEndian.PutUint32(buf[4:8], totalLen)
	// pb head length 流式为0
	binary.BigEndian.PutUint16(buf[8:10], uint16(0))
	// StreamID
	binary.BigEndian.PutUint32(buf[10:14], frameHead.StreamID)
	// stream buf 无字节序
	copy(buf[16:], streamBuf)
	return buf, nil
}

// encodeCloseFrame  服务端流式编码Close 帧
func (s *ServerStreamCodec) encodeCloseFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	defer s.deleteInitMeta(msg)
	closeFrame, ok := msg.StreamFrame().(*TrpcStreamCloseMeta)
	if !ok {
		return nil, errEncodeCloseFrame
	}
	msg.WithStreamID(frameHead.StreamID)
	streamBuf, err := proto.Marshal(closeFrame)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeDataFrame 服务端流式编码data帧
func (s *ServerStreamCodec) encodeDataFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	return frameWrite(frameHead, reqbuf)
}

// encodeInitFrame 服务端流式编码Init 帧
func (s *ServerStreamCodec) encodeInitFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	rsp := getStreamInitMeta(msg)
	rsp.ContentType = uint32(msg.SerializationType())
	rsp.ContentEncoding = uint32(msg.CompressType())
	rspMeta := &TrpcStreamInitResponseMeta{}
	if e := msg.ServerRspErr(); e != nil {
		rspMeta.Ret = e.Code
		rspMeta.ErrorMsg = []byte(e.Msg)
	}
	rsp.ResponseMeta = rspMeta
	streamBuf, err := proto.Marshal(rsp)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// encodeFeedbackFrame  服务端流式编码feedback帧
func (s *ServerStreamCodec) encodeFeedbackFrame(frameHead *FrameHead, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	feedback, ok := msg.StreamFrame().(*TrpcStreamFeedBackMeta)
	if !ok {
		return nil, errEncodeFeedbackFrame
	}
	streamBuf, err := proto.Marshal(feedback)
	if err != nil {
		return nil, err
	}
	return frameWrite(frameHead, streamBuf)
}

// getStreamInitMeta 从msg里面获取TrpcStreamInitMeta，不存在就创建新的
func getStreamInitMeta(msg codec.Msg) *TrpcStreamInitMeta {
	rsp, ok := msg.StreamFrame().(*TrpcStreamInitMeta)
	if !ok {
		rsp = &TrpcStreamInitMeta{ResponseMeta: &TrpcStreamInitResponseMeta{}}
	}
	return rsp
}

// getFrameHead 获取FrameHead ，如果msg里面没有，则创建新的
func getFrameHead(msg codec.Msg) *FrameHead {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok {
		frameHead = &FrameHead{}
	}
	return frameHead
}

// Encode trpc服务端流式编码入口
func (s *ServerStreamCodec) Encode(msg codec.Msg, reqbuf []byte) (rspbuf []byte, err error) {
	frameHead := getFrameHead(msg)
	switch TrpcStreamFrameType(frameHead.StreamFrameType) {
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return s.encodeInitFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return s.encodeDataFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return s.encodeCloseFrame(frameHead, msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return s.encodeFeedbackFrame(frameHead, msg, reqbuf)
	default:
		return nil, errUnknownFrameType
	}
}

// Decode trpc服务端流式解码，解出包头和流式帧数据
func (s *ServerStreamCodec) Decode(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	frameHead := &FrameHead{
		FrameType:       reqbuf[2],
		StreamFrameType: reqbuf[3],
		StreamID:        binary.BigEndian.Uint32(reqbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(reqbuf[14:16]),
	}
	msg.WithFrameHead(frameHead)
	switch TrpcStreamFrameType(frameHead.StreamFrameType) {
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_INIT:
		return s.decodeInitFrame(msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_DATA:
		return s.decodeDataFrame(msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_CLOSE:
		return s.decodeCloseFrame(msg, reqbuf)
	case TrpcStreamFrameType_TRPC_STREAM_FRAME_FEEDBACK:
		return s.decodeFeedbackFrame(msg, reqbuf)
	default:
		return nil, errUnknownFrameType
	}
}

// decodeFeedbackFrame decode出feedback帧
func (s *ServerStreamCodec) decodeFeedbackFrame(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok {
		return nil, errFrameHeadTypeInvalid
	}
	streamID := frameHead.StreamID
	msg.WithStreamID(streamID)
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	feedback := &TrpcStreamFeedBackMeta{}
	if err := proto.Unmarshal(reqbuf[frameHeadLen:], feedback); err != nil {
		return nil, err
	}
	msg.WithStreamFrame(feedback)
	return nil, nil
}

// setInitMeta 找到对应的initMeta，并设置相应ServerRPCName，用来映射对应的handler
func (s *ServerStreamCodec) setInitMeta(msg codec.Msg) error {
	streamID := msg.StreamID()
	addr := msg.RemoteAddr()
	s.m.RLock()
	defer s.m.RUnlock()
	if streamIDToInitMeta, ok := s.initMetas[addr]; ok {
		if initMeta, ok := streamIDToInitMeta[streamID]; ok {
			msg.WithServerRPCName(string(initMeta.GetRequestMeta().GetFunc()))
			return nil
		}
	}
	return errUninitializedMeta
}

// deleteInitMeta 清理掉initMeta的缓存信息
func (s *ServerStreamCodec) deleteInitMeta(msg codec.Msg) {
	addr := msg.RemoteAddr()
	streamID := msg.StreamID()
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.initMetas[addr], streamID)
	if len(s.initMetas[addr]) == 0 {
		delete(s.initMetas, addr)
	}
}

// decodeCloseFrame trpc流式 解码Close帧
func (s *ServerStreamCodec) decodeCloseFrame(msg codec.Msg, rspbuf []byte) ([]byte, error) {
	frameHead := getFrameHead(msg)
	streamID := frameHead.StreamID
	msg.WithStreamID(streamID)
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	close := &TrpcStreamCloseMeta{}
	if err := proto.Unmarshal(rspbuf[frameHeadLen:], close); err != nil {
		return nil, err
	}
	// 如果Close类型为Reset 或者Ret不为0，则将错误返回给服务端
	if close.GetCloseType() == int32(TrpcStreamCloseType_TRPC_STREAM_RESET) || close.GetRet() != 0 {
		e := &errs.Error{
			Type: errs.ErrorTypeCalleeFramework,
			Code: close.GetRet(),
			Desc: "trpc",
			Msg:  string(close.GetMsg()),
		}
		msg.WithServerRspErr(e)
	}
	msg.WithStreamFrame(close)
	return nil, nil
}

// decodeDataFrame 服务端流式解码数据帧
func (s *ServerStreamCodec) decodeDataFrame(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	frameHead, ok := msg.FrameHead().(*FrameHead)
	if !ok {
		return nil, errFrameHeadTypeInvalid
	}
	streamID := frameHead.StreamID
	msg.WithStreamID(streamID)
	if err := s.setInitMeta(msg); err != nil {
		return nil, err
	}
	reqBody := reqbuf[frameHeadLen:]
	return reqBody, nil
}

// decodeInitFrame 服务端解码Init帧
func (s *ServerStreamCodec) decodeInitFrame(msg codec.Msg, reqbuf []byte) ([]byte, error) {
	initMeta := &TrpcStreamInitMeta{}
	if err := proto.Unmarshal(reqbuf[frameHeadLen:], initMeta); err != nil {
		return nil, err
	}
	frameHead := &FrameHead{
		FrameType:       reqbuf[2],
		StreamFrameType: reqbuf[3],
		StreamID:        binary.BigEndian.Uint32(reqbuf[10:14]),
		FrameReserved:   binary.BigEndian.Uint16(reqbuf[14:16]),
	}
	msg.WithStreamID(frameHead.StreamID)
	s.updateMsg(msg, frameHead, initMeta)
	s.storeInitMeta(msg, initMeta)
	msg.WithStreamFrame(initMeta)
	return nil, nil
}

// storeInitMeta 存储InitMeta，每次获取到新的数据帧的时候要带上
func (s *ServerStreamCodec) storeInitMeta(msg codec.Msg, initMeta *TrpcStreamInitMeta) {
	streamID := msg.StreamID()
	addr := msg.RemoteAddr()
	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.initMetas[addr]; ok {
		s.initMetas[addr][streamID] = initMeta
	} else {
		t := make(map[uint32]*TrpcStreamInitMeta)
		t[streamID] = initMeta
		s.initMetas[addr] = t
	}
}

// updateMsg 服务端流式更新Msg
func (s *ServerStreamCodec) updateMsg(msg codec.Msg, frameHead *FrameHead, initMeta *TrpcStreamInitMeta) {
	msg.WithFrameHead(frameHead)
	// 设置具体业务协议请求包头
	req := initMeta.GetRequestMeta()

	// 设置上游服务名
	msg.WithCallerServiceName(string(req.GetCaller()))
	msg.WithCalleeServiceName(string(req.GetCallee()))
	// 设置当前请求的rpc方法名(命令字)
	msg.WithServerRPCName(string(req.GetFunc()))
	// 设置body的序列化方式
	msg.WithSerializationType(int(initMeta.GetContentType()))
	// 设置body压缩方式
	msg.WithCompressType(int(initMeta.GetContentEncoding()))
	msg.WithDyeing((req.GetMessageType() & uint32(TrpcMessageType_TRPC_DYEING_MESSAGE)) != 0)

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
}
