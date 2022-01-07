// Package overloadctrl 定义了过载保护接口。
package overloadctrl

import (
	"context"
)

// OverloadController 定义了过载保护接口。
type OverloadController interface {
	Acquire(ctx context.Context, addr string) (Token, error)
}

// Token 定义了过载保护返回的 token 的接口。
type Token interface {
	OnResponse(ctx context.Context, err error)
}

// NoopOC 是 OverloadController 的空实现。
type NoopOC struct{}

// Acquire 总是放行，并返回一个空 Token。
func (NoopOC) Acquire(context.Context, string) (Token, error) {
	return NoopToken{}, nil
}

// NoopToken 是 Token 的空实现。
type NoopToken struct{}

// OnResponse 什么都不做。
func (NoopToken) OnResponse(context.Context, error) {}
