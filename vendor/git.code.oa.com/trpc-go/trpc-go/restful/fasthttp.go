package restful

import (
	"bytes"
	"context"
	"unsafe"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
)

// FastHTTPHeaderMatcher 从 fasthttp request 头映射到 tRPC Stub Context
type FastHTTPHeaderMatcher func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	serviceName, methodName string,
) (context.Context, error)

// 默认的 FastHTTPHeaderMatcher
var defaultFastHTTPHeaderMatcher = func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	serviceName, methodName string,
) (context.Context, error) {
	return withNewMessage(ctx, serviceName, methodName), nil
}

// FastHTTPRespHandler fasthttp 用户自定义的回包处理
type FastHTTPRespHandler func(
	ctx context.Context,
	requestCtx *fasthttp.RequestCtx,
	resp proto.Message,
	body []byte,
) error

// defaultFastHTTPRespHandler 默认 fasthttp 回包处理
func defaultFastHTTPRespHandler(stubCtx context.Context, requestCtx *fasthttp.RequestCtx,
	protoResp proto.Message, body []byte) error {
	// 压缩
	writer := requestCtx.Response.BodyWriter()
	// fasthttp 暂不支持 header 一个 key 下获取 multi values，
	// ctx.Request.Header.Peek 相当于原生 net/http 里的 req.Header.Get
	_, c := compressorForTranscoding(
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerContentEncoding))},
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerAcceptEncoding))},
	)
	if c != nil {
		writeCloser, err := c.Compress(writer)
		if err != nil {
			return err
		}
		defer writeCloser.Close()
		requestCtx.Response.Header.Set(headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	// 设置响应码
	statusCode := GetStatusCodeOnSucceed(stubCtx)
	requestCtx.SetStatusCode(statusCode)

	// 设置 body
	if statusCode != fasthttp.StatusNoContent && statusCode != fasthttp.StatusNotModified {
		writer.Write(body)
	}

	return nil
}

// bytes2str 高性能 []byte 转 string
func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// HandleRequestCtx fasthttp handler
func (r *Router) HandleRequestCtx(ctx *fasthttp.RequestCtx) {
	newCtx := context.Background()
	for _, tr := range r.transcoders[bytes2str(ctx.Method())] {
		fieldValues, err := tr.pat.Match(bytes2str(ctx.Path()))
		if err == nil {
			// 头转换
			stubCtx, err := r.opts.FastHTTPHeaderMatcher(newCtx, ctx,
				r.opts.ServiceName, tr.name)
			if err != nil {
				r.opts.FastHTTPErrHandler(stubCtx, ctx, errs.New(errs.RetServerDecodeFail, err.Error()))
				return
			}

			// 获取请求 Compressor 和 Serializer
			// fasthttp 暂不支持 header 一个 key 下获取 multi values，
			// ctx.Request.Header.Peek 相当于原生 net/http 里的 req.Header.Get
			reqCompressor, respCompressor := compressorForTranscoding(
				[]string{bytes2str(ctx.Request.Header.Peek(headerContentEncoding))},
				[]string{bytes2str(ctx.Request.Header.Peek(headerAcceptEncoding))},
			)
			reqSerializer, respSerializer := serializerForTranscoding(
				[]string{bytes2str(ctx.Request.Header.Peek(headerContentType))},
				[]string{bytes2str(ctx.Request.Header.Peek(headerAccept))},
			)

			// 获取 query params
			form := make(map[string][]string)
			ctx.QueryArgs().VisitAll(func(key []byte, value []byte) {
				form[bytes2str(key)] = append(form[bytes2str(key)], bytes2str(value))
			})

			// 设置转码参数
			params := paramsPool.Get().(*transcodeParams)
			params.reqCompressor = reqCompressor
			params.respCompressor = respCompressor
			params.reqSerializer = reqSerializer
			params.respSerializer = respSerializer
			params.body = bytes.NewBuffer(ctx.PostBody())
			params.fieldValues = fieldValues
			params.form = form

			// 转码
			resp, body, err := tr.transcode(stubCtx, params)
			if err != nil {
				fastHTTPErrorHandler(stubCtx, ctx, err)
				putBackCtxMessage(stubCtx)
				putBackParams(params)
				return
			}

			// response content-type 设置
			ctx.Response.Header.Set(headerContentType, respSerializer.ContentType())

			// 回包
			if err := r.opts.FastHTTPRespHandler(stubCtx, ctx, resp, body); err != nil {
				r.opts.FastHTTPErrHandler(stubCtx, ctx, errs.New(errs.RetServerEncodeFail, err.Error()))
			}
			putBackCtxMessage(stubCtx)
			putBackParams(params)
			return
		}
	}
	r.opts.FastHTTPErrHandler(newCtx, ctx, errs.New(errs.RetServerNoFunc, "failed to match any pattern"))
}
