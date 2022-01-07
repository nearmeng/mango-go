// Package sample 数据采样
package sample

import (
	"math/rand"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

const (
	hundred = 100
)

// isClose 采样开关
func isClose(rate int) bool {
	if rate >= hundred || rate <= 0 {
		return true
	}
	return false
}

// Pass 是否需要采样
func Pass(rate int) bool {
	if isClose(rate) {
		return true
	}
	// 采样率判断
	if rate > rand.Intn(hundred) {
		return true
	}

	return false
}

// Reduction 采样数据还原
func Reduction(rate int, statLogList *nmnt.StatLogList) {
	if isClose(rate) {
		return
	}

	for _, log := range statLogList.Log {
		for _, content := range log.Content {
			for _, value := range content.Values {
				value.Count = value.Count * hundred / int32(rate)
				switch value.Policy {
				case nmnt.Policy_SUM:
					fallthrough
				case nmnt.Policy_AVG:
					value.Value = value.Value * hundred / float64(rate)
				default:
				}
			}
		}
	}
}
