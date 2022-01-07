package http

import (
	"context"
	"sync"
	"time"
)

// valueDetachedCtx 在保证 ctx timeout/cancel 传递性的同时，卸除所有与 ctx 相关联的 value。
// 在原 ctx timeout/cancel 后，valueDetachedCtx 必须释放原 ctx，保证与原 ctx 相关联的资源能够被正常 GC。
type valueDetachedCtx struct {
	mu  sync.Mutex
	ctx context.Context
}

// detachCtxValue 从 ctx 创建一个新的 valueDetachedCtx。
func detachCtxValue(ctx context.Context) context.Context {
	if ctx.Done() == nil {
		return context.Background()
	}
	c := valueDetachedCtx{ctx: ctx}
	go func() {
		<-ctx.Done()
		deadline, ok := ctx.Deadline()
		c.mu.Lock()
		c.ctx = &ctxRemnant{
			deadline:    deadline,
			hasDeadline: ok,
			err:         ctx.Err(),
			done:        ctx.Done(),
		}
		c.mu.Unlock()
	}()
	return &c
}

// Deadline 实现 Context 的 Deadline 方法。
func (c *valueDetachedCtx) Deadline() (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Deadline()
}

// Done 实现 Context 的 Done 方法。
func (c *valueDetachedCtx) Done() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx.Done()
}

// Err 实现 Context 的 Err 方法。
func (c *valueDetachedCtx) Err() error {
	c.mu.Lock()
	c.mu.Unlock()
	return c.ctx.Err()
}

// Value 总是返回 nil。
func (c *valueDetachedCtx) Value(_ interface{}) interface{} {
	return nil
}

// ctxRemnant 是 valueDetachedCtx 在 timeout/cancel 后的残迹，保存原 ctx 的部分信息，保证原 ctx 能够被正常 GC。
type ctxRemnant struct {
	deadline    time.Time
	hasDeadline bool
	err         error
	done        <-chan struct{}
}

// Deadline 返回保存的 deadline 信息。
func (c *ctxRemnant) Deadline() (time.Time, bool) {
	return c.deadline, c.hasDeadline
}

// Done 返回保存的 Done channel。
func (c *ctxRemnant) Done() <-chan struct{} {
	return c.done
}

// Err 返回保存的错误。
func (c *ctxRemnant) Err() error {
	return c.err
}

// Value 总是返回 nil。
func (c *ctxRemnant) Value(_ interface{}) interface{} {
	return nil
}
