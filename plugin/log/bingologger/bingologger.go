// Package bingologger bingo default logger.
/*
1. Supports synchronous write back and asynchronous write back
2. Support for splitting across days and according to file size
*/
package bingologger

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	sysruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/spf13/viper"
)

func init() {
	plugin.RegisterPluginFactory(&factory{})
}

// factory log factory.
type factory struct{}

// LogCfg log cfg.
type LogCfg struct {
	LogPath           string `mapstructure:"path"`
	LogLevel          int32  `mapstructure:"level"`             // log level [Hot update support]
	FileSplitMB       int32  `mapstructure:"filesplitmb"`       // MB file split according to the xxx.log.N[Hot update support].
	IsAsync           bool   `mapstructure:"isasync"`           // async logger.
	AsyncCacheSize    int    `mapstructure:"asynccachesize"`    // The maximum bytes to be cached, need to be actively written once.
	AsyncWriteMillSec int    `mapstructure:"asyncwritemillsec"` // Timed write back time interval, in milliseconds.
}

// Type plugin type.
func (f *factory) Type() string {
	return "log"
}

// Name plugin name.
func (f *factory) Name() string {
	return "bingologger"
}

// Setup create logger.
func (f *factory) Setup(v *viper.Viper) (interface{}, error) {
	var cfg LogCfg

	fmt.Printf("log_path is %s\n", v.GetString("path"))
	//if err := mapstructure.Decode(c, &cfg); err != nil {
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	logger := &BingoLogger{}
	if err := logger.Init(cfg); err != nil {
		return nil, fmt.Errorf("Setup err:%w", err)
	}
	log.SetLogger(logger)
	log.Info("Init log path=%s, level=%d", cfg.LogPath, cfg.LogLevel)
	return logger, nil
}

// Destory sync not close.
func (f *factory) Destroy(i interface{}) error {
	logger, ok := i.(*BingoLogger)
	if logger == nil || !ok {
		return errors.New("Destory BingoLogger type assert")
	}

	if err := logger.Sync(); err != nil {
		return fmt.Errorf("Destory BingoLogger err:%w", err)
	}

	log.Info("Destory logger only exec sync")
	return nil
}

// Reload reload config.
func (f *factory) Reload(i interface{}, c map[string]interface{}) error {
	var cfg LogCfg
	if err := mapstructure.Decode(c, &cfg); err != nil {
		return err
	}

	if err := CheckCfgValid(cfg); err != nil {
		return fmt.Errorf("Reload err:%w", err)
	}

	l, ok := i.(*BingoLogger)
	if !ok {
		return fmt.Errorf("Reload type assert type=%T", i)
	}

	atomic.StoreInt32(&l.level, cfg.LogLevel)
	atomic.StoreInt32(&l.fileSplitMB, cfg.FileSplitMB)

	log.Debug("Logger Reload success, level=%d fileSplitMB=%d", cfg.LogLevel, cfg.FileSplitMB)
	return nil
}

const _asyncByteSizePerIOWrite = 10 << 20 // Maximum number of bytes per write when asynchronous

// BingoLogger bingo default log plugin.
type BingoLogger struct {
	level             int32              // log level.
	fileName          string             // file path.
	fileSplitMB       int32              // file split size(MB).
	isAsync           bool               // async log.
	asyncWriteMillSec int                // tick write mill second.
	logTime           int64              // last log time.
	fileFd            *os.File           // cur file handle.
	fileCreateTime    time.Time          // cur file create time.
	generation        int                // file split count.
	isInited          bool               // bingo logger inited.
	lock              sync.Mutex         // write lock.
	bufChan           chan []byte        // log buf channel queue.
	ntfChan           chan chan struct{} // Notification of immediate write or closing of write concurrent.
}

// NewBingoLogger new logger with LogCfg.
func NewBingoLogger(cfg LogCfg) (*BingoLogger, error) {
	logger := &BingoLogger{}
	if err := logger.Init(cfg); err != nil {
		return nil, fmt.Errorf("Setup err:%w", err)
	}
	return logger, nil
}

// Init init logger with cfg.
func (l *BingoLogger) Init(cfg LogCfg) error {
	if l.isInited {
		return errors.New("Init BingoLogger has inited")
	}

	if err := CheckCfgValid(cfg); err != nil {
		return fmt.Errorf("Init err:%w", err)
	}

	l.fileName = cfg.LogPath
	l.level = cfg.LogLevel
	l.isAsync = cfg.IsAsync
	l.asyncWriteMillSec = cfg.AsyncWriteMillSec
	l.fileSplitMB = cfg.FileSplitMB

	if cfg.IsAsync {
		l.bufChan = make(chan []byte, cfg.AsyncCacheSize)
		l.ntfChan = make(chan chan struct{})
		if err := l.asyncWriteLoop(); err != nil {
			return fmt.Errorf("init err:%w", err)
		}
	}

	l.isInited = true
	return nil
}

// CheckCfgValid 检查参数有效性.
func CheckCfgValid(cfg LogCfg) error {
	if len(cfg.LogPath) == 0 {
		return errors.New("CheckCfgValid LogPath empty")
	}
	if cfg.LogLevel < 0 {
		return errors.New("CheckCfgValid Level need >= 0")
	}

	if cfg.FileSplitMB <= 0 {
		return errors.New("CheckCfgValid FileSplitMB need > 0")
	}

	if cfg.IsAsync {
		if cfg.AsyncCacheSize <= 0 {
			return errors.New("CheckCfgValid AsyncCacheSize need > 0")
		}
		if cfg.AsyncWriteMillSec <= 0 {
			return errors.New("CheckCfgValid AsyncWriteMillSec need > 0")
		}
	}
	return nil
}

// GetLevel get log level.
func (l *BingoLogger) GetLevel() int {
	return int(atomic.LoadInt32(&l.level))
}

// SetLevel set log level.
func (l *BingoLogger) SetLevel(level int) {
	if level > log.LogLevelFatal {
		level = log.LogLevelFatal
	}
	if level < 0 {
		level = 0
	}
	l.level = int32(level)
}

// Output output log.
func (l *BingoLogger) Output(depth int, logType string, a log.AcntLogger, format string, v ...interface{}) {
	// time
	now := time.Now()
	year, month, day := now.Date()
	hour, min, sec := now.Clock()

	// content
	logContent := fmt.Sprintf(format, v...)

	// file and function name
	file, line := getCallPath(depth)
	if l.isInited { // nolint
		// Fprintf replace fprintf no bytes to string convert
		if a != nil {
			_, _ = fmt.Fprintf(l, "[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] [%s:%d] %s %s\n",
				year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
				logType, file, line,
				a.GetLogStr(), logContent)
		} else {
			_, _ = fmt.Fprintf(l, "[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] [%s:%d] %s\n",
				year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
				logType, file, line,
				logContent)
		}
	} else {
		if a != nil {
			_, _ = fmt.Printf("[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] [%s:%d] %s %s",
				year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
				logType, file, line,
				a.GetLogStr(), logContent)
		} else {
			_, _ = fmt.Printf("[%04d-%02d-%02d %02d:%02d:%02d.%03d] [%s] [%s:%d] %s",
				year, int(month), day, hour, min, sec, now.Nanosecond()/1e6, // nolint
				logType, file, line,
				logContent)
		}
	}
}

// Write write bytes.
func (l *BingoLogger) Write(buf []byte) (n int, err error) {
	if !l.isInited {
		return len(buf), nil
	}

	if l.isAsync {
		l.writeAsync(buf)
		return len(buf), nil
	}
	return l.writeSync(buf)
}

// Sync Write log to file immediately.
func (l *BingoLogger) Sync() error {
	if !l.isInited {
		return nil
	}
	if !l.isAsync {
		return nil
	}
	// ntf and wait result
	doneChan := make(chan struct{})
	l.ntfChan <- doneChan
	<-doneChan
	return nil
}

// writeSync Synchronized log writing.
func (l *BingoLogger) writeSync(buf []byte) (n int, err error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.updateFileFd(); err != nil {
		return 0, fmt.Errorf("Write err:%w", err)
	}

	now := time.Now()
	l.logTime = now.Unix()
	if l.fileFd == nil {
		return 0, errors.New("log file not opened")
	}
	return l.fileFd.Write(buf)
}

// writeSync Asynchronous log writing.
func (l *BingoLogger) writeAsync(buf []byte) {
	// Put it in the buffer channel first, and then write it in bulk,
	// buf can not be cached, you need to do a copy, because the underlying fprintf is multiplexed buf.
	newBuf := make([]byte, len(buf))
	copy(newBuf, buf)
	select {
	case l.bufChan <- newBuf:
	default:
		{
			// When the buff is full, you have to notify the writing coordinator to write it immediately.
			// Avoid this process by adjusting the cache parameters.
			l.ntfChan <- nil
			// Re-deliver, ensure that the logs are not lost.
			l.bufChan <- newBuf
		}
	}
}

// asyncWriteLoop Asynchronous log writing concurrently.
//nolint:revive
func (l *BingoLogger) asyncWriteLoop() error {
	loopFunc := func() {
		sendBuffer := bytes.NewBuffer(make([]byte, 0, _asyncByteSizePerIOWrite))

		// writeAllFunc Write all buf's of the current moment to the file.
		writeAllFunc := func() {
			bufLen := len(l.bufChan)
			if bufLen > 0 {
				for i := 0; i < bufLen; i++ {
					buf := <-l.bufChan
					// Write logs of no more than a certain number of bytes each time to avoid reallocation of buffer.
					if sendBuffer.Len()+len(buf) > _asyncByteSizePerIOWrite {
						_, _ = l.writeSync(sendBuffer.Bytes())
						sendBuffer.Reset()
					}
					_, _ = sendBuffer.Write(buf)
				}

				if sendBuffer.Len() > 0 {
					_, _ = l.writeSync(sendBuffer.Bytes())
					sendBuffer.Reset()
				}
			}
		}

		tickTimer := time.NewTicker(time.Duration(l.asyncWriteMillSec) * time.Millisecond)

		for {
			select {
			case doneChan, ok := <-l.ntfChan:
				{
					// Write Immediately
					writeAllFunc()
					if doneChan != nil {
						doneChan <- struct{}{}
					}
					if !ok {
						// Close write concurrent
						break
					}
				}
			case <-tickTimer.C:
				{
					// Timed write back
					writeAllFunc()
				}
			}
		}
	}

	loopFunc()
	return nil
}

func (l *BingoLogger) updateFileFd() error {
	if len(l.fileName) == 0 {
		return errors.New("updateFileFd filename is empty")
	}

	if err := l.updateOldFileFd(); err != nil {
		return fmt.Errorf("updateFileFd err:%w", err)
	}

	if l.fileFd == nil {
		if err := l.openLogFile(l.fileName); err != nil {
			return fmt.Errorf("updateFileFd err:%w", err)
		}
	}
	return nil
}

// updateOldFileFd Whether the currently open file should be closed and renamed.
func (l *BingoLogger) updateOldFileFd() error {
	if l.fileFd == nil {
		return nil
	}

	now := time.Now()
	fi, err := os.Stat(l.fileName)
	// After deleting the log file during operation, need to repair the file handle.
	if err != nil {
		if os.IsNotExist(err) {
			l.fileFd = nil
			return nil
		}
		return fmt.Errorf("updateOldFileFd os.Stat filename=%s err:%w", l.fileName, err)
	}

	// Testing across days.
	if l.fileCreateTime.Unix()+86400 < now.Unix() || l.fileCreateTime.Day() != now.Day() {
		if err := l.moveLogFile(l.fileName, l.fileCreateTime); err != nil {
			return fmt.Errorf("updateOldFileFd moveLogFile filename=%s err:%w", l.fileName, err)
		}
		// reset split count.
		l.generation = 0
		return nil
	}

	// File size detection, oversized files should be split.
	if fi.Size() >= int64(atomic.LoadInt32(&l.fileSplitMB))<<20 {
		if err := l.moveLogFile(l.fileName, l.fileCreateTime); err != nil {
			return fmt.Errorf("updateOldFileFd moveLogFile filename=%s err:%w", l.fileName, err)
		}
		// increase split count.
		l.generation++
		return nil
	}
	return nil
}

func getFileCreateTime(filePath string) (time.Time, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	statT, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, errors.New("sys stat")
	}
	return time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec), nil
}

func (l *BingoLogger) openLogFile(filePath string) error {
	if l.fileFd != nil {
		return errors.New("log file have opened")
	}
	if dir := path.Dir(filePath); len(dir) != 0 {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
	}

	var perm fs.FileMode = 0o666
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, perm)
	if nil != err {
		return err
	}

	fileCreateTime, err := getFileCreateTime(filePath)
	if err != nil {
		return err
	}

	l.fileCreateTime = fileCreateTime
	l.fileFd = fd
	return nil
}

func (l *BingoLogger) moveLogFile(filePath string, createTime time.Time) error {
	// Close the old log file first.
	if l.fileFd != nil {
		_ = l.fileFd.Close()
		l.fileFd = nil
	}
	// Generate new filenames based on creation time and split serial number.
	var newFilePath string

	for {
		if l.generation == 0 {
			newFilePath = fmt.Sprintf("%s.%02d%02d%02d", filePath, createTime.Year(), createTime.Month(), createTime.Day())
		} else {
			newFilePath = fmt.Sprintf("%s.%02d%02d%02d.%d",
				filePath, createTime.Year(), createTime.Month(), createTime.Day(), l.generation)
		}

		// Need to find a non-existent file to move.
		isExist, err := l.isLogFileExist(newFilePath)
		if err != nil {
			return fmt.Errorf("moveLogFile newFilePath=%s err:%w", newFilePath, err)
		}
		if !isExist {
			break
		}
		l.generation++
	}

	return os.Rename(filePath, newFilePath)
}

func (l *BingoLogger) isLogFileExist(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// getCallPath 获取调用的 包名/文件名 行数.
func getCallPath(depth int) (string, int) {
	const callOffset = 1
	_, file, line, ok := sysruntime.Caller(depth + callOffset)
	if !ok {
		return "", 0
	}

	idx := strings.LastIndexByte(file, '/')
	if idx == -1 {
		return file, line
	}
	idx = strings.LastIndexByte(file[:idx], '/')
	if idx == -1 {
		return file, line
	}

	return file[idx+1:], line
}
