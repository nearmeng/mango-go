// Package ring 提供并发安全的环形队列，支持多读多写
package ring

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/cpu"
)

const (
	// cacheLinePadSize CPU Cacheline的大小
	cacheLinePadSize = unsafe.Sizeof(cpu.CacheLinePad{})
)

var (
	// ErrQueueFull 队列已满
	ErrQueueFull = errors.New("queue is full")
)

type ringItem struct {
	putSeq uint32 // 期待写入的序列号
	getSeq uint32 // 期待读出的序列号
	value  interface{}
	_      [cacheLinePadSize - 8 - 16]byte
}

// Ring 提供并发安全的环形队列，基于Disruptor思想实现(简化实现)
// https://lmax-exchange.github.io/disruptor/disruptor.html
type Ring struct {
	capacity uint32 // 环形队列的容量,包括Empty元素,必须为2的幂
	mask     uint32 // 环形队列容量掩码
	_        [cacheLinePadSize - 8]byte
	head     uint32 // 最近已读元素的序列号
	_        [cacheLinePadSize - 4]byte
	tail     uint32 // 最近已写元素的序列号
	_        [cacheLinePadSize - 4]byte
	data     []ringItem // 队列元素
	_        [cacheLinePadSize - unsafe.Sizeof([]ringItem{})]byte
}

// New 创建环形队列
func New(capacity uint32) *Ring {
	capacity = roundUpToPower2(capacity)
	if capacity < 2 {
		capacity = 2
	}

	r := &Ring{
		capacity: capacity,
		mask:     capacity - 1,
		data:     make([]ringItem, capacity),
	}
	// 初始化每个槽位期待的读写序列号
	for i := range r.data {
		r.data[i].getSeq = uint32(i)
		r.data[i].putSeq = uint32(i)
	}
	// 起始从Index为1开始填包
	r.data[0].getSeq = capacity
	r.data[0].putSeq = capacity
	return r
}

// Put 向环形队列中添加元素，队列满时直接返回
func (r *Ring) Put(val interface{}) error {
	// 获取写序列号使用权
	seq, err := r.acquirePutSequence()
	if err != nil {
		return err
	}
	// 写入元素
	r.commit(seq, val)
	return nil
}

// Get 从环形队列中获取一个元素，返回元素值和剩余元素个数
func (r *Ring) Get() (interface{}, uint32) {
	// 获取读序列号使用权
	head, size, left := r.acquireGetSequence(1)
	if size == 0 {
		return nil, 0
	}
	// 读出元素
	return r.consume(head), left
}

// Gets 从环形队列中获取元素并append到v中，返回获取和剩余元素个数
func (r *Ring) Gets(val *[]interface{}) (uint32, uint32) {
	// 批量获取读序列号使用权
	head, size, left := r.acquireGetSequence(uint32(cap(*val) - len(*val)))
	if size == 0 {
		return 0, 0
	}
	// 批量读出元素
	for seq, i := head, uint32(0); i < size; seq, i = seq+1, i+1 {
		*val = append(*val, r.consume(seq))
	}
	return size, left
}

// Cap 获取环形队列可容纳元素的数量
func (r *Ring) Cap() uint32 {
	// 队列占用一个槽位表示empty,真实容量为mask
	return r.mask
}

// Size 获取环形队列当前元素个数
func (r *Ring) Size() uint32 {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return r.quantity(head, tail)
}

// IsEmpty 判断队列是否为空
func (r *Ring) IsEmpty() bool {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return head == tail
}

// IsFull 判断队列是否已满
func (r *Ring) IsFull() bool {
	head := atomic.LoadUint32(&r.head)
	next := atomic.LoadUint32(&r.tail) + 1
	return next-head > r.mask
}

// String 打印Ring结构
func (r *Ring) String() string {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return fmt.Sprintf("Ring：Cap=%v, Head=%v, Tail=%v, Size=%v\n",
		r.Cap(), head, tail, r.Size())
}

func (r *Ring) quantity(head, tail uint32) uint32 {
	return tail - head
}

func (r *Ring) acquirePutSequence() (uint32, error) {
	var tail, head, next uint32
	mask := r.mask
	for {
		head = atomic.LoadUint32(&r.head)
		tail = atomic.LoadUint32(&r.tail)
		next = tail + 1
		left := r.quantity(head, next)
		// 队列已满，直接返回
		if left > mask {
			return 0, ErrQueueFull
		}
		// 获取到序列号，直接返回
		if atomic.CompareAndSwapUint32(&r.tail, tail, next) {
			return next, nil
		}
		// 未抢到序列号,让出CPU等待一会儿再抢，提升抢占命中率，减少CPU空转
		runtime.Gosched()
	}
}

func (r *Ring) acquireGetSequence(ask uint32) (uint32, uint32, uint32) {
	var tail, head, size uint32
	for {
		head = atomic.LoadUint32(&r.head)
		tail = atomic.LoadUint32(&r.tail)
		left := r.quantity(head, tail)
		// 队列为空，直接返回
		if left < 1 {
			return head, 0, 0
		}
		size = left
		if ask < left {
			size = ask
		}
		// 获取到序列号，直接返回
		if atomic.CompareAndSwapUint32(&r.head, head, head+size) {
			return head + 1, size, left - size
		}
		// 未抢到序列号,让出CPU等待一会儿再抢，提升抢占命中率，减少CPU空转
		runtime.Gosched()
	}
}

func (r *Ring) commit(seq uint32, val interface{}) {
	item := &r.data[seq&r.mask]
	for {
		getSeq := atomic.LoadUint32(&item.getSeq)
		putSeq := atomic.LoadUint32(&item.putSeq)
		// 等待数据可写就绪。由于获取序列号使用权和读写数据操作分离
		// 存在短暂时间旧数据还未读出，等待读操作完成并置位getSeq
		if seq == putSeq && getSeq == putSeq {
			break
		}
		runtime.Gosched()
	}
	// 完成写操作并设置putSeq为下次期待的写序列号
	item.value = val
	atomic.AddUint32(&item.putSeq, r.capacity)
}

func (r *Ring) consume(seq uint32) interface{} {
	item := &r.data[seq&r.mask]
	for {
		getSeq := atomic.LoadUint32(&item.getSeq)
		putSeq := atomic.LoadUint32(&item.putSeq)
		// 等待数据可读就绪。由于获取序列号使用权和读写数据操作分离
		// 存在短暂时间写数据还未写完，等待写操作完成并置位putSeq
		if seq == getSeq && getSeq == (putSeq-r.capacity) {
			break
		}
		runtime.Gosched()
	}
	// 完成读操作并设置getSeq为下次期待的读序列号
	val := item.value
	item.value = nil
	atomic.AddUint32(&item.getSeq, r.capacity)
	return val
}

// roundUpToPower2 整数向上取整到2的N次方
func roundUpToPower2(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
