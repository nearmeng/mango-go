package pcgmonitor

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// logLevel SDK内部日志级别
type logLevel int

const (
	levelNil logLevel = iota
	levelTrace
	levelDebug
	levelInfo
	levelWarn
	levelError
)

var levelNames = map[logLevel]string{
	levelNil:   "unknown",
	levelTrace: "trace",
	levelDebug: "debug",
	levelInfo:  "info",
	levelWarn:  "warn",
	levelError: "error",
}

const (
	attaSep = '|'
)

// reportLog SDK内部日志上报，关键日志上报远端
func (s *Instance) reportLog(level logLevel, needNative bool, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if needNative {
		log.Println(msg)
	}

	c := s.remoteConfig()
	if !c.attaInfo.LogOpen {
		return
	}

	// 按日志格式拼接数据
	content := s.logContent(level, msg)

	ret := attaAPI.SendString(c.attaInfo.LogAttaID, c.attaInfo.LogAttaToken, content)
	if ret != 0 {
		log.Printf("trpc_report_api_go: log SendString fail, ret:%d", ret)
	}
}

// logContent 日志完整内容
func (s *Instance) logContent(level logLevel, msg string) string {
	var buf strings.Builder
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05")) // 时间
	buf.WriteByte(attaSep)
	buf.WriteString(levelNames[level]) // 级别
	buf.WriteByte(attaSep)
	if s.svrInfoType == svrInfoFrame {
		buf.WriteString(s.frameSvrInfo.App + "." + s.frameSvrInfo.Server) // app.server
		buf.WriteByte(attaSep)
		buf.WriteString(s.frameSvrInfo.IP) // IP
		buf.WriteByte(attaSep)
		buf.WriteString(s.frameSvrInfo.Container) // 容器
		buf.WriteByte(attaSep)
		buf.WriteString(s.frameSvrInfo.FrameCode) // 框架标识
	} else {
		buf.WriteString(s.commSvrInfo.CommName) // app.server
		buf.WriteByte(attaSep)
		buf.WriteString(s.commSvrInfo.IP) // IP
		buf.WriteByte(attaSep)
		buf.WriteString(s.commSvrInfo.Container) // 容器
		buf.WriteByte(attaSep)
		buf.WriteString(commReportFrameCode) // 框架标识
	}
	buf.WriteByte(attaSep)
	buf.WriteString(language) // sdk语言
	buf.WriteByte(attaSep)
	buf.WriteString(version()) // sdk版本
	buf.WriteByte(attaSep)
	buf.WriteString(msg) // content
	return buf.String()
}
