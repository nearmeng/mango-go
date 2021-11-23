package log

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// defaultLogger Used for logging before the initialization of the logging plugin, print console and write to tmp file.
// Use process name as log file name, log file and process in the same directory.
type defaultLogger struct {
	level  int
	fileFd *os.File
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{}
}

// Output Output Log.
func (l *defaultLogger) Output(depth int, logType string, a AcntLogger, format string, v ...interface{}) {
	// time
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()

	// content
	logContent := fmt.Sprintf(format, v...)

	var logStr string
	if a != nil { // nolint
		logStr = fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] %s %s",
			year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
			logType, a.GetLogStr(), logContent)
	} else {
		logStr = fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] %s",
			year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
			logType, logContent)
	}

	_, _ = fmt.Fprintln(l, logStr)
	_, _ = fmt.Println(logStr)
}

// Write Write bytes.
func (l *defaultLogger) Write(buf []byte) (n int, err error) {
	fd := l.getFileFd()
	if fd == nil {
		return 0, errors.New("Write filefd is nil")
	}
	return fd.Write(buf)
}

// Sync sync cache to file.
func (l *defaultLogger) Sync() error {
	if fd := l.getFileFd(); fd != nil {
		return fd.Sync()
	}
	return nil
}

// SetLevel set log level.
func (l *defaultLogger) SetLevel(level int) {
	l.level = level
}

// GetLevel get log level.
func (l *defaultLogger) GetLevel() int {
	return l.level
}

// getFileFd get or open file handle.
func (l *defaultLogger) getFileFd() *os.File {
	if l.fileFd != nil {
		return l.fileFd
	}

	// Use process name as log file name, log file and process in the same directory.
	defaultTmpLogFile := fmt.Sprintf("%s_default.log", os.Args[0])

	var perm fs.FileMode = 0o666
	fd, err := os.OpenFile(defaultTmpLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, perm)
	if err != nil {
		return nil
	}
	l.fileFd = fd
	return fd
}
