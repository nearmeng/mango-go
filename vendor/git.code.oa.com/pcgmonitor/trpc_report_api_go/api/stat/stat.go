// Package stat 007上报数据
package stat

import (
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

const (
	ignore = "ignore"
)

// MultiDimensionsFix 多维度上报，维度值修复，增加对维度值的数量限制，维度值强置为ignore，预防恶意上报
func MultiDimensionsFix(logName string, dimensions []string, details map[string]*config.Detail,
	initOffset int) {
	detail, ok := details[logName]
	if !ok || len(detail.Ignore) == 0 {
		return
	}

	maxIndex := len(dimensions) - 1
	for _, index := range detail.Ignore {
		realIndex := index - initOffset
		if realIndex >= 0 && realIndex <= maxIndex {
			dimensions[realIndex] = ignore
		}
	}
}

// SingleDimensionFix 单维度上报，纬度值修复，预防恶意上报
func SingleDimensionFix(logName string, dimension string, details map[string]*config.Detail, initOffset int) string {
	result := dimension
	detail, ok := details[logName]
	if !ok || len(detail.Ignore) == 0 {
		return result
	}

	for _, index := range detail.Ignore {
		realIndex := index - initOffset
		if realIndex >= 0 {
			result = ignore
			break
		}
	}
	return result
}

// ValueFix 上报数值强制为1(历史原因，字段已暴露)，避免外部设置
func ValueFix(statValues []*nmnt.StatValue) {
	for _, v := range statValues {
		v.Count = 1 // count强制为1，内部汇集使用
	}
}
