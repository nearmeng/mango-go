// Package log 提供框架和应用日志输出能力
package log

import (
	"context"
	"fmt"
	"os"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/internal/env"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var traceEnabled = traceEnableFromEnv()

// 读取环境变量,判断是否开启Trace
// 默认关闭
// 为空或者为0，关闭Trace
// 非空且非0，开启Trace
func traceEnableFromEnv() bool {
	switch os.Getenv(env.LogTrace) {
	case "":
		fallthrough
	case "0":
		return false
	default:
		return true
	}
}

// EnableTrace 开启trace级别日志
func EnableTrace() {
	traceEnabled = true
}

// SetLevel 设置不同的输出对应的日志级别, output为输出数组下标 "0" "1" "2"
func SetLevel(output string, level Level) {
	GetDefaultLogger().SetLevel(output, level)
}

// GetLevel 获取不同输出对应的日志级别
func GetLevel(output string) Level {
	return GetDefaultLogger().GetLevel(output)
}

// WithFields 设置一些业务自定义数据到每条log里:比如uid，imei等, fields 必须kv成对出现
func WithFields(fields ...string) Logger {
	return GetDefaultLogger().WithFields(fields...)
}

// WithFieldsContext 以当前 context logger 为基础，增加设置一些业务自定义数据到每条log里:比如uid，imei等, fields 必须kv成对出现
func WithFieldsContext(ctx context.Context, fields ...string) Logger {
	logger, ok := codec.Message(ctx).Logger().(Logger)
	if !ok {
		return WithFields(fields...)
	}
	return logger.WithFields(fields...)
}

// RedirectStdLog 重定向标准库 log 的输出到 trpc logger 的 info 级别日志中
// 重定向后，log flag 为 0，prefix 为空
// 它返回的函数可以用来恢复 log flag 和 prefix，并将输出重定向到 os.Stderr
func RedirectStdLog(logger Logger) (func(), error) {
	return RedirectStdLogAt(logger, zap.InfoLevel)
}

// RedirectStdLogAt 重定向标准库 log 的输出到 trpc logger 指定级别的日志中
// 重定向后，log flag 为 0，prefix 为空
// 它返回的函数可以用来恢复 log flag 和 prefix，并将输出重定向到 os.Stderr
func RedirectStdLogAt(logger Logger, level zapcore.Level) (func(), error) {
	if l, ok := logger.(*zapLog); ok {
		return zap.RedirectStdLogAt(l.logger, level)
	}
	if l, ok := logger.(*ZapLogWrapper); ok {
		return zap.RedirectStdLogAt(l.l.logger, level)
	}
	return nil, fmt.Errorf("log: only supports redirecting std logs to trpc zap logger")
}

// Trace logs to TRACE log. Arguments are handled in the manner of fmt.Print.
func Trace(args ...interface{}) {
	if traceEnabled {
		GetDefaultLogger().Trace(args...)
	}
}

// Tracef logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, args ...interface{}) {
	if traceEnabled {
		GetDefaultLogger().Tracef(format, args...)
	}
}

// TraceContext logs to TRACE log. Arguments are handled in the manner of fmt.Print.
func TraceContext(ctx context.Context, args ...interface{}) {
	if !traceEnabled {
		return
	}
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Trace(args...)
			return
		}
		l.l.Trace(args...)
	case Logger:
		l.Trace(args...)
	default:
		GetDefaultLogger().Trace(args...)
	}
}

// TraceContextf logs to TRACE log. Arguments are handled in the manner of fmt.Printf.
func TraceContextf(ctx context.Context, format string, args ...interface{}) {
	if !traceEnabled {
		return
	}
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Tracef(format, args...)
			return
		}
		l.l.Tracef(format, args...)
	case Logger:
		l.Tracef(format, args...)
	default:
		GetDefaultLogger().Tracef(format, args...)
	}
}

// Debug logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
func Debug(args ...interface{}) {
	GetDefaultLogger().Debug(args...)
}

// Debugf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

// Info logs to INFO log. Arguments are handled in the manner of fmt.Print.
func Info(args ...interface{}) {
	GetDefaultLogger().Info(args...)
}

// Infof logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

// Warn logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func Warn(args ...interface{}) {
	GetDefaultLogger().Warn(args...)
}

// Warnf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

// Error logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func Error(args ...interface{}) {
	GetDefaultLogger().Error(args...)
}

// Errorf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

// Fatal logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func Fatal(args ...interface{}) {
	GetDefaultLogger().Fatal(args...)
}

// Fatalf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

// WithContextFields 设置一些业务自定义数据到每条 log 里:比如 uid、imei 等, fields 必须成对出现。
//
// 如果 ctx 中已经设置了 Msg，该函数会返回原始 ctx，否则，它会返回一个新的 ctx。
func WithContextFields(ctx context.Context, fields ...string) context.Context {
	ctx, msg := codec.EnsureMessage(ctx)
	logger, ok := msg.Logger().(Logger)
	if ok && logger != nil {
		logger = logger.WithFields(fields...)
	} else {
		logger = GetDefaultLogger().WithFields(fields...)
	}
	msg.WithLogger(logger)
	return ctx
}

// DebugContext logs to DEBUG log. Arguments are handled in the manner of fmt.Print.
func DebugContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Debug(args...)
			return
		}
		l.l.Debug(args...)
	case Logger:
		l.Debug(args...)
	default:
		GetDefaultLogger().Debug(args...)
	}
}

// DebugContextf logs to DEBUG log. Arguments are handled in the manner of fmt.Printf.
func DebugContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Debugf(format, args...)
			return
		}
		l.l.Debugf(format, args...)
	case Logger:
		l.Debugf(format, args...)
	default:
		GetDefaultLogger().Debugf(format, args...)
	}
}

// InfoContext logs to INFO log. Arguments are handled in the manner of fmt.Print.
func InfoContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Info(args...)
			return
		}
		l.l.Info(args...)
	case Logger:
		l.Info(args...)
	default:
		GetDefaultLogger().Info(args...)
	}
}

// InfoContextf logs to INFO log. Arguments are handled in the manner of fmt.Printf.
func InfoContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Infof(format, args...)
			return
		}
		l.l.Infof(format, args...)
	case Logger:
		l.Infof(format, args...)
	default:
		GetDefaultLogger().Infof(format, args...)
	}
}

// WarnContext logs to WARNING log. Arguments are handled in the manner of fmt.Print.
func WarnContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Warn(args...)
			return
		}
		l.l.Warn(args...)
	case Logger:
		l.Warn(args...)
	default:
		GetDefaultLogger().Warn(args...)
	}
}

// WarnContextf logs to WARNING log. Arguments are handled in the manner of fmt.Printf.
func WarnContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Warnf(format, args...)
			return
		}
		l.l.Warnf(format, args...)
	case Logger:
		l.Warnf(format, args...)
	default:
		GetDefaultLogger().Warnf(format, args...)
	}
}

// ErrorContext logs to ERROR log. Arguments are handled in the manner of fmt.Print.
func ErrorContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Error(args...)
			return
		}
		l.l.Error(args...)
	case Logger:
		l.Error(args...)
	default:
		GetDefaultLogger().Error(args...)
	}
}

// ErrorContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func ErrorContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Errorf(format, args...)
			return
		}
		l.l.Errorf(format, args...)
	case Logger:
		l.Errorf(format, args...)
	default:
		GetDefaultLogger().Errorf(format, args...)
	}
}

// FatalContext logs to ERROR log. Arguments are handled in the manner of fmt.Print.
// that all Fatal logs will exit with os.Exit(1).
// Implementations may also call os.Exit() with a non-zero exit code.
func FatalContext(ctx context.Context, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Fatal(args...)
			return
		}
		l.l.Fatal(args...)
	case Logger:
		l.Fatal(args...)
	default:
		GetDefaultLogger().Fatal(args...)
	}
}

// FatalContextf logs to ERROR log. Arguments are handled in the manner of fmt.Printf.
func FatalContextf(ctx context.Context, format string, args ...interface{}) {
	switch l := codec.Message(ctx).Logger().(type) {
	case *ZapLogWrapper:
		// 保护 l 或者 l.l 不可为空
		if l == nil || l.l == nil {
			GetDefaultLogger().Fatalf(format, args...)
			return
		}
		l.l.Fatalf(format, args...)
	case Logger:
		l.Fatalf(format, args...)
	default:
		GetDefaultLogger().Fatalf(format, args...)
	}
}
