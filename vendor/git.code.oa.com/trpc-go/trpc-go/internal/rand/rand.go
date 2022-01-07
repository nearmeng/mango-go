// Package rand 提供公共协程安全的随机函，代码有参考grpc随机函数，相比grpc提供了种子外部调用，并且随机函数当作类来处理
package rand

import (
	"math/rand"
	"sync"
)

// SafeRand 安全随机函数结构
type SafeRand struct {
	r  *rand.Rand
	mu sync.Mutex
}

// NewSafeRand 新建一个种子对应可产生随机算法的SafeRand
func NewSafeRand(seed int64) *SafeRand {
	c := &SafeRand{
		r: rand.New(rand.NewSource(seed)),
	}
	return c
}

// Int63n 提供int64协程安全的随机函数
func (c *SafeRand) Int63n(n int64) int64 {
	c.mu.Lock()
	res := c.r.Int63n(n)
	c.mu.Unlock()
	return res
}

// Intn 提供int协程安全的随机函数
func (c *SafeRand) Intn(n int) int {
	c.mu.Lock()
	res := c.r.Intn(n)
	c.mu.Unlock()
	return res
}

// Float64 提供float64协程安全的随机函数
func (c *SafeRand) Float64() float64 {
	c.mu.Lock()
	res := c.r.Float64()
	c.mu.Unlock()
	return res
}
