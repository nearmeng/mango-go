package restful

import "git.code.oa.com/trpc-go/trpc-go/internal/httprule"

// Pattern 对外暴露 httprule.PathTemplate
type Pattern struct {
	*httprule.PathTemplate
}

// Parse 提供给 tRPC 工具使用，检查 url path 是否合法
func Parse(urlPath string) (*Pattern, error) {
	tpl, err := httprule.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	return &Pattern{tpl}, nil
}

// Enforce 保证 url path 是合法的前提下返回一个 Pattern
func Enforce(urlPath string) *Pattern {
	pattern, err := Parse(urlPath)
	if err != nil {
		panic(err)
	}
	return pattern
}
