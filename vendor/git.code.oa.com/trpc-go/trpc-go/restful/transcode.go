package restful

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/restful/dat"
	"google.golang.org/protobuf/proto"
)

const (
	// 默认的 http req body buffer 大小
	defaultBodyBufferSize = 4096
)

// transcoder tRPC 和 httpjson 转码器
type transcoder struct {
	name                 string
	input                func() ProtoMessage
	output               func() ProtoMessage
	handler              HandleFunc
	httpMethod           string
	pat                  *Pattern
	body                 BodyLocator
	respBody             ResponseBodyLocator
	router               *Router
	dat                  *dat.DoubleArrayTrie
	discardUnknownParams bool
}

// transcodeParams transcode 所需参数
type transcodeParams struct {
	reqCompressor  Compressor
	respCompressor Compressor
	reqSerializer  Serializer
	respSerializer Serializer
	body           io.Reader
	fieldValues    map[string]string
	form           url.Values
}

// transcodeParams 池
var paramsPool = sync.Pool{
	New: func() interface{} {
		return &transcodeParams{}
	},
}

// 把 transcodeParams 放回池中
func putBackParams(params *transcodeParams) {
	params.reqCompressor = nil
	params.respCompressor = nil
	params.reqSerializer = nil
	params.respSerializer = nil
	params.body = nil
	params.fieldValues = nil
	params.form = nil
	paramsPool.Put(params)
}

// transcode tRPC 和 httpjson 转码
func (tr *transcoder) transcode(
	stubCtx context.Context,
	params *transcodeParams,
) (proto.Message, []byte, error) {
	// 初始化 tRPC 请求
	protoReq := tr.input()
	protoResp := tr.output()

	// 先 body 转码
	if err := tr.transcodeBody(protoReq, params.body, params.reqCompressor,
		params.reqSerializer); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// 根据请求路径中匹配到的 fieldValues 转码
	if err := tr.transcodeFieldValues(protoReq, params.fieldValues); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// 根据 query params 转码
	if err := tr.transcodeQueryParams(protoReq, params.form); err != nil {
		return nil, nil, errs.New(errs.RetServerDecodeFail, err.Error())
	}

	// tRPC Stub 处理
	if err := tr.handle(stubCtx, protoReq, protoResp); err != nil {
		return nil, nil, err
	}

	// 回包
	// HttpRule.response_body 只指定了序列化的字段，所以先不做压缩，留给用户自定义
	buf, err := tr.transcodeResp(protoResp, params.respSerializer)
	if err != nil {
		return nil, nil, errs.New(errs.RetServerEncodeFail, err.Error())
	}
	return protoResp, buf, nil
}

// http request body buffer 池
var bodyBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, defaultBodyBufferSize))
	},
}

// transcodeBody 根据 http request body 转码
func (tr *transcoder) transcodeBody(protoReq proto.Message, body io.Reader, c Compressor, s Serializer) error {
	// HttpRule body 未设置
	if tr.body == nil {
		return nil
	}

	// 解压缩
	var reader io.Reader
	var err error
	if c != nil {
		if reader, err = c.Decompress(body); err != nil {
			return fmt.Errorf("failed to decompress request body: %w", err)
		}
	} else {
		reader = body
	}

	// 读取 body
	buffer := bodyBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bodyBufferPool.Put(buffer)
	if _, err := io.Copy(buffer, reader); err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// 反序列化
	if err := s.Unmarshal(buffer.Bytes(), tr.body.Locate(protoReq)); err != nil {
		return fmt.Errorf("failed to unmarshal req body: %w", err)
	}

	// patch 方法还要尝试更新 field mask
	if tr.httpMethod == "PATCH" && tr.body.Body() != "*" {
		return setFieldMask(protoReq.ProtoReflect(), tr.body.Body())
	}

	return nil
}

// transcodeFieldValues 根据请求路径中匹配到的 fieldValues 转码
func (tr *transcoder) transcodeFieldValues(msg proto.Message, fieldValues map[string]string) error {
	for fieldPath, value := range fieldValues {
		if err := PopulateMessage(msg, strings.Split(fieldPath, "."), []string{value}); err != nil {
			return err
		}
	}
	return nil
}

// transcodeQueryParams 根据 http query params 转码
func (tr *transcoder) transcodeQueryParams(msg proto.Message, form url.Values) error {
	// 若 HttpRule body 为 * 则忽略查询参数
	if tr.body != nil && tr.body.Body() == "*" {
		return nil
	}

	for key, values := range form {
		// 过滤已被 HttpRule pattern 和 body 引用的
		if tr.dat != nil && tr.dat.CommonPrefixSearch(strings.Split(key, ".")) {
			continue
		}
		// 填充 proto message
		if err := PopulateMessage(msg, strings.Split(key, "."), values); err != nil {
			if !tr.discardUnknownParams || !errors.Is(err, ErrTraverseNotFound) {
				return err
			}
		}
	}

	return nil
}

// handle tRPC Stub 处理
func (tr *transcoder) handle(ctx context.Context, reqbody, rspbody interface{}) error {
	filters := tr.router.opts.FilterFunc()
	serviceImpl := tr.router.opts.ServiceImpl
	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return tr.handler(serviceImpl, ctx, reqbody, rspbody)
	}
	return filters.Handle(ctx, reqbody, rspbody, handleFunc)
}

// transcodeResp 回包转码
func (tr *transcoder) transcodeResp(protoResp proto.Message, s Serializer) ([]byte, error) {
	// 序列化
	var obj interface{}
	if tr.respBody == nil {
		obj = protoResp
	} else {
		obj = tr.respBody.Locate(protoResp)
	}
	return s.Marshal(obj)
}
