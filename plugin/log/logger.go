// Package log level interface logging framework.
package log

// _globalLogger bingo logger must have one.
var (
	_globalLogger Logger = newDefaultLogger()
	_normalDepth         = 2
)

const (
	_traceTag = "TRACE"
	_debugTag = "DEBUG"
	_infoTag  = "INFO"
	_errorTag = "ERR"
	_fatalTag = "FATAL"
)

// SetLogger set default logger.
func SetLogger(logger Logger) {
	_globalLogger = logger
}

// Sync Blocking interface that writes the cached logs to file immediately,
// usually called when the service is shut down.
func Sync() error {
	if _globalLogger == nil {
		return nil
	}
	return _globalLogger.Sync()
}

// Trace trace log.
func Trace(format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if _globalLogger.GetLevel() > LogLevelTrace {
		return
	}
	TraceCnt.Add(1)

	_globalLogger.Output(_normalDepth, _traceTag, nil, format, v...)
}

// Debug debug log.
func Debug(format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if _globalLogger.GetLevel() > LogLevelDebug {
		return
	}
	DebugCnt.Add(1)
	_globalLogger.Output(_normalDepth, _debugTag, nil, format, v...)
}

// Info info log.
func Info(format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if _globalLogger.GetLevel() > LogLevelInfo {
		return
	}
	InfoCnt.Add(1)
	_globalLogger.Output(_normalDepth, _infoTag, nil, format, v...)
}

// Error err log.
func Error(format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if _globalLogger.GetLevel() > LogLevelError {
		return
	}
	ErrorCnt.Add(1)
	_globalLogger.Output(_normalDepth, _errorTag, nil, format, v...)
}

// Fatal fatal log.
func Fatal(format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if _globalLogger.GetLevel() > LogLevelFatal {
		return
	}
	FatalCnt.Add(1)
	_globalLogger.Output(_normalDepth, _fatalTag, nil, format, v...)
}

// AcntTrace trace log.
func AcntTrace(a AcntLogger, format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if !a.InLogWhitelist() {
		if _globalLogger.GetLevel() > LogLevelTrace {
			return
		}
	}

	TraceCnt.Add(1)
	_globalLogger.Output(_normalDepth, _traceTag, a, format, v...)
}

// AcntDebug debug log.
func AcntDebug(a AcntLogger, format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if !a.InLogWhitelist() {
		if _globalLogger.GetLevel() > LogLevelDebug {
			return
		}
	}

	DebugCnt.Add(1)
	_globalLogger.Output(_normalDepth, _debugTag, a, format, v...)
}

// AcntInfo info log.
func AcntInfo(a AcntLogger, format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if !a.InLogWhitelist() {
		if _globalLogger.GetLevel() > LogLevelInfo {
			return
		}
	}

	InfoCnt.Add(1)
	_globalLogger.Output(_normalDepth, _infoTag, a, format, v...)
}

// AcntError err log.
func AcntError(a AcntLogger, format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if !a.InLogWhitelist() {
		if _globalLogger.GetLevel() > LogLevelError {
			return
		}
	}

	ErrorCnt.Add(1)
	_globalLogger.Output(_normalDepth, _errorTag, a, format, v...)
}

// AcntFatal fatal log.
func AcntFatal(a AcntLogger, format string, v ...interface{}) {
	if _globalLogger == nil {
		return
	}

	if !a.InLogWhitelist() {
		if _globalLogger.GetLevel() > LogLevelFatal {
			return
		}
	}

	FatalCnt.Add(1)
	_globalLogger.Output(_normalDepth, _fatalTag, a, format, v...)
}

func GetLogLevel() int {
	if _globalLogger == nil {
		return LogLevelNull
	}
	return _globalLogger.GetLevel()
}
