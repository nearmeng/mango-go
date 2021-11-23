package log

import (
	"fmt"
)

// MockSetDefaultLogger 设置mock日志.
func MockSetDefaultLogger() {
	SetLogger(&MockLogger{})
}

// MockLogger mock日志.
type MockLogger struct{}

// Output mock日志输出.
func (*MockLogger) Output(depth int, logType string, a AcntLogger, format string, v ...interface{}) {
	_, _ = fmt.Println(fmt.Sprintf(format, v...))
}

// SetLevel mock日志设置等级.
func (*MockLogger) SetLevel(level int) {}

// GetLevel 获取mock日志等级.
func (*MockLogger) GetLevel() int {
	return -1
}

// Write 写mock日志.
func (*MockLogger) Write(buf []byte) (n int, err error) {
	return len(buf), nil
}

// Sync 同步mock日志.
func (*MockLogger) Sync() error {
	return nil
}
