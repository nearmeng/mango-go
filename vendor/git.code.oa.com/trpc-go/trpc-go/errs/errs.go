// Package errs trpc错误码类型，里面包含errcode errmsg，多语言通用
package errs

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

var (
	traceable bool
	content   string
)

// trpc return code
const (
	RetOK = 0

	RetServerDecodeFail   = 1
	RetServerEncodeFail   = 2
	RetServerNoService    = 11
	RetServerNoFunc       = 12
	RetServerTimeout      = 21
	RetServerOverload     = 22
	RetServerThrottled    = 23
	RetServerSystemErr    = 31
	RetServerAuthFail     = 41
	RetServerValidateFail = 51

	RetClientTimeout         = 101
	RetClientConnectFail     = 111
	RetClientEncodeFail      = 121
	RetClientDecodeFail      = 122
	RetClientThrottled       = 123
	RetClientRouteErr        = 131
	RetClientNetErr          = 141
	RetClientValidateFail    = 151
	RetClientCanceled        = 161
	RetClientStreamQueueFull = 201

	RetUnknown = 999
)

// Err 框架错误值
var (
	ErrOK error

	ErrServerNoService       = NewFrameError(RetServerNoService, "server router no service")
	ErrServerNoFunc          = NewFrameError(RetServerNoFunc, "server router no rpc method")
	ErrServerTimeout         = NewFrameError(RetServerTimeout, "server message handle timeout")
	ErrServerOverload        = NewFrameError(RetServerOverload, "server overload")
	ErrServerRoutinePoolBusy = NewFrameError(RetServerOverload, "server goroutine pool too small")
	ErrServerClose           = NewFrameError(RetServerSystemErr, "server close")

	ErrServerNoResponse = NewFrameError(RetOK, "server no response")
	ErrClientNoResponse = NewFrameError(RetOK, "client no response")

	ErrUnknown = NewFrameError(RetUnknown, "unknown error")
)

// 跳过的堆栈帧数
var stackSkip = defaultStackSkip

const (
	defaultStackSkip = 3
)

// ErrorType 错误码类型 包括框架错误码和业务错误码
const (
	ErrorTypeFramework       = 1
	ErrorTypeBusiness        = 2
	ErrorTypeCalleeFramework = 3 // client调用返回的错误码，代表是下游框架错误码
)

// Success 成功提示字符串
const (
	Success = "success"
)

// Error 错误码结构 包含 错误码类型 错误码 错误信息
type Error struct {
	Type int
	Code int32
	Msg  string
	Desc string

	st []uintptr // 调用栈
}

// Error 实现error接口，返回error描述
func (e *Error) Error() string {
	if e == nil {
		return Success
	}
	switch e.Type {
	case ErrorTypeFramework:
		return fmt.Sprintf("type:framework, code:%d, msg:%s", e.Code, e.Msg)
	case ErrorTypeCalleeFramework:
		return fmt.Sprintf("type:callee framework, code:%d, msg:%s", e.Code, e.Msg)
	default:
		return fmt.Sprintf("type:business, code:%d, msg:%s", e.Code, e.Msg)
	}
}

// Format 实现fmt.Formatter接口
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, e.Error())
			for _, pc := range e.st {
				f := errors.Frame(pc)
				str := fmt.Sprintf("\n%+v", f)
				if !isOutput(str) {
					continue
				}
				_, _ = io.WriteString(s, str)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	default:
		// unknown format
		_, _ = fmt.Fprintf(s, "%%!%c(errs.Error=%s)", verb, e.Error())
	}
}

// SetTraceable 控制error是否带堆栈跟踪
func SetTraceable(x bool) {
	traceable = x
}

// SetTraceableWithContent 控制error是否带堆栈跟踪，打印堆栈信息时，根据content进行过滤。
// 避免输出大量无用信息。可以通过配置content为服务名的方式，过滤掉其他插件的堆栈信息。
func SetTraceableWithContent(c string) {
	traceable = true
	content = c
}

// SetStackSkip 支持设置跳过的堆栈帧数,在封装 New 方法时，可以将 stackSkip 设为 4 （根据封装层数确定）
// 此函数用于在项目启动前设置，不保证并发安全
func SetStackSkip(skip int) {
	stackSkip = skip
}

func isOutput(str string) bool {
	return strings.Contains(str, content)
}

func callers() []uintptr {
	var pcs [32]uintptr
	n := runtime.Callers(stackSkip, pcs[:])
	st := pcs[0:n]
	return st
}

// New 创建一个error，默认为业务错误类型，提高业务开发效率
func New(code int, msg string) error {
	err := &Error{
		Type: ErrorTypeBusiness,
		Code: int32(code),
		Msg:  msg,
	}
	if traceable {
		err.st = callers()
	}
	return err
}

// Newf 创建一个error，默认为业务错误类型，msg支持格式化字符串
func Newf(code int, format string, params ...interface{}) error {
	msg := fmt.Sprintf(format, params...)
	return New(code, msg)
}

// NewFrameError 创建一个框架error
func NewFrameError(code int, msg string) error {
	err := &Error{
		Type: ErrorTypeFramework,
		Code: int32(code),
		Msg:  msg,
		Desc: "trpc",
	}
	if traceable {
		err.st = callers()
	}
	return err
}

// Code 通过error获取error code
func Code(e error) int {
	if e == nil {
		return 0
	}
	err, ok := e.(*Error)
	if !ok {
		return RetUnknown
	}
	if err == (*Error)(nil) {
		return 0
	}
	return int(err.Code)
}

// Msg 通过error获取error msg
func Msg(e error) string {
	if e == nil {
		return Success
	}
	err, ok := e.(*Error)
	if !ok {
		return e.Error()
	}
	if err == (*Error)(nil) {
		return Success
	}
	return err.Msg
}
