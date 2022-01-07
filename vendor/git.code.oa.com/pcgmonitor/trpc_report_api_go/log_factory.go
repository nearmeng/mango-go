package pcgmonitor

import "log"

type m007Logger struct {
	Inst *Instance
}

func (m *m007Logger) Trace(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      TraceLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) TraceWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      TraceLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) Debug(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      DebugLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) DebugWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      DebugLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) Info(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      InfoLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) InfoWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      InfoLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) Warning(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      WarnLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) WarningWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      WarnLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) Error(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      ErrorLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) ErrorWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      WarnLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) Fatal(logName, content string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		Level:      FatalLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) FatalWithContextId(logName, content, contextId string, args ...string) {
	params := &HawkLogParams{
		Name:       logName,
		Content:    content,
		ContextID:  contextId,
		Level:      FatalLevel,
		Dimensions: args,
	}
	m.sendLog(params)
}

func (m *m007Logger) sendLog(params *HawkLogParams) {
	err := m.Inst.ReportHawkLog(params)
	if err != nil {
		log.Printf("%v", err)
	}
}
