package log

import (
	"github.com/nearmeng/mango-go/plugin/log/protolog"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Log level.
const (
	LogLevelNull  = 0
	LogLevelTrace = 1
	LogLevelDebug = 2
	LogLevelInfo  = 3
	LogLevelError = 4
	LogLevelFatal = 5
)

// log count.
var (
	TraceCnt atomic.Int32
	DebugCnt atomic.Int32
	InfoCnt  atomic.Int32
	ErrorCnt atomic.Int32
	FatalCnt atomic.Int32
)

// AcntLogger log AcntLogger.
type AcntLogger interface {
	InLogWhitelist() bool
	GetLogStr() string
}

// Logger interface.
type Logger interface {
	// write log like syslog.
	Output(depth int, logType string, a AcntLogger, format string, v ...interface{})
	// io.Writer interface.
	Write(buf []byte) (n int, err error)
	// sync flush buffered log, need call before exit.
	Sync() error
	SetLevel(level int)
	GetLevel() int
}

// FormatPB returns the message value as a string,
// which is the message serialized in the protobuf text format.
func FormatPB(m protoreflect.ProtoMessage) string {
	return protolog.MarshalOptions{}.Format(m)
}
