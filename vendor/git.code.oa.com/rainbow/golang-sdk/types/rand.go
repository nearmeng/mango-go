package types

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Rand 不全局共用rand库，减少锁竞争
type Rand struct {
	Seed int64
	Pool *sync.Pool
}

var (
	mrand        = NewRand()
	seq   uint64 = 0
)

// NewRand 初始化随机数发生器
func NewRand() *Rand {
	p := &sync.Pool{New: func() interface{} {
		return rand.New(rand.NewSource(getSeed()))
	},
	}
	mrand := &Rand{
		Pool: p,
	}
	return mrand
}

// 获取种子
func getSeed() int64 {
	seed := (atomic.AddUint64(&seq, 1)%1000 + 1000) * 1e15
	tn := time.Now().UnixNano() % 1e15
	return int64(seed) + tn
}

func (s *Rand) getrand() *rand.Rand {
	return s.Pool.Get().(*rand.Rand)
}
func (s *Rand) putrand(r *rand.Rand) {
	s.Pool.Put(r)
}

// Intn 获取随机数
func (s *Rand) Intn(n int) int {
	r := s.getrand()
	defer s.putrand(r)

	return r.Intn(n)
}

// Int63 non-negative
func (s *Rand) Int63() int64 {
	r := s.getrand()
	defer s.putrand(r)
	return r.Int63()
}

// Int63n <=n
func Int63n(n int64) int64 {
	r := mrand.getrand()
	defer mrand.putrand(r)
	return r.Int63n(n)

}
