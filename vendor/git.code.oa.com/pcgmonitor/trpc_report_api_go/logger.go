package pcgmonitor

type Logger interface {
	/**
	日志方法,除必填字段外,args都默认为日志的维度，相同业务为了查询方便 维度应该相同
	*/
	Trace(logName, content string, args ...string)
	TraceWithContextId(logName, content, contextId string, args ...string)
	Debug(logName, content string, args ...string)
	DebugWithContextId(logName, content, contextId string, args ...string)
	Info(logName, content string, args ...string)
	InfoWithContextId(logName, content, contextId string, args ...string)
	Warning(logName, content string, args ...string)
	WarningWithContextId(logName, content, contextId string, args ...string)
	Error(logName, content string, args ...string)
	ErrorWithContextId(logName, content, contextId string, args ...string)
	Fatal(logName, content string, args ...string)
	FatalWithContextId(logName, content, contextId string, args ...string)
}

var defaultLogger Logger

func init() {
	logger := &m007Logger{Inst: defaultInstance}
	defaultLogger = logger
}

// Trace 日志  args 为日志维度 相同日志名称 日志唯独要一样,方便页面查询
func Trace(logName, content string, args ...string) {
	defaultLogger.Trace(logName, content, args...)
}

// Trace 日志  args 为日志维度 相同日志名称 日志唯独要一样,方便页面查询
func TraceWithContextId(logName, content, contextId string, args ...string) {
	defaultLogger.TraceWithContextId(logName, content, contextId, args...)
}

func Info(logName, content string, args ...string) {
	defaultLogger.Info(logName, content, args...)
}

func InfoWithContextId(logName, content, contextId string, args ...string) {
	defaultLogger.InfoWithContextId(logName, content, contextId, args...)
}

func Warning(logName, content string, args ...string) {
	defaultLogger.Warning(logName, content, args...)
}

func WarningWithContextId(logName, content, contextId string, args ...string) {
	defaultLogger.WarningWithContextId(logName, content, contextId, args...)
}

func Error(logName, content string, args ...string) {
	defaultLogger.Error(logName, content, args...)
}

func ErrorWithContextId(logName, content, contextId string, args ...string) {
	defaultLogger.ErrorWithContextId(logName, content, contextId, args...)
}

func Fatal(logName, content string, args ...string) {
	defaultLogger.Fatal(logName, content, args...)
}

func FatalWithContextId(logName, content, contextId string, args ...string) {
	defaultLogger.FatalWithContextId(logName, content, contextId, args...)
}
