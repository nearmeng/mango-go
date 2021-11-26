package cache

import (
	"container/list"
	"sync"
	"time"
)

const (
	DefaultCapacity = 1024        // 默认容量
	DefaultAge      = time.Minute // 默认有效时间
)

// entry in cache
type entry struct {
	value    interface{}   // 数据
	lruNode  *list.Element // 最近访问节点
	ageNode  *list.Element // 时长节点
	loadTime time.Time     // 加载时刻
	size     int           // 大小
}

// Cache: cache m by key
type Cache struct {
	sync.RWMutex
	m        map[interface{}]*entry // id -> entry
	lruList  *list.List             // LRU链表(key)
	ageList  *list.List             // 超时链表(key)
	hits     int                    // 命中数
	misses   int                    // 未命中数
	capacity int                    // 缓存上限
	ttl      time.Duration          // 生存时间
	size     int64                  // 大小
}

// New: create a cache
// capacity limited to cap, age is entry available duration
func New() *Cache {
	return &Cache{
		m:        make(map[interface{}]*entry),
		lruList:  list.New(),
		ageList:  list.New(),
		capacity: DefaultCapacity,
		ttl:      DefaultAge,
	}
}

// GetCapacity 获取cache的容量
func (c *Cache) GetCapacity() int {
	return c.capacity
}

// SetCapacity 设置cache的容量
func (c *Cache) SetCapacity(cap int) {
	if cap > 1 {
		c.capacity = cap
	}
}

// SetTTL 设置cache的超时时间
func (c *Cache) SetTTL(d time.Duration) {
	c.ttl = d
}

// Hits: get cache hits count
func (c *Cache) Hits() int {
	return c.hits
}

// Misses: get cache misses count
func (c *Cache) Misses() int {
	return c.misses
}

// Size: get mem size
func (c *Cache) Size() int64 {
	return c.size
}

// Len: get cached number
func (c *Cache) Len() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.m)
}

// Get: find a entry by key
func (c *Cache) Get(k interface{}) interface{} {
	c.Lock()
	defer c.Unlock()

	// 先清除过期元素
	c.removeExpired()

	// update the last access list
	if v, ok := c.m[k]; ok {
		v.lruNode = c.lruList.PushFront(c.lruList.Remove(v.lruNode))
		c.hits++
		return v.value
	}

	c.misses++
	return nil
}

// Update 更新某个存在的key的value
func (c *Cache) Update(k interface{}, v interface{}) interface{} {
	c.Lock()
	defer c.Unlock()

	// 先清除过期元素
	c.removeExpired()

	// update
	if value, ok := c.m[k]; ok {
		value.lruNode = c.lruList.PushFront(c.lruList.Remove(value.lruNode))
		c.hits++
		value.value = v
		return value.value
	}

	c.misses++

	return nil
}

// Add: add a key & entry & size
func (c *Cache) AddWithSize(k interface{}, v interface{}, s int) bool {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.m[k]; ok {
		return false
	}

	c.cull() // cull if necessary

	c.m[k] = &entry{
		value:    v,
		lruNode:  c.lruList.PushFront(k),
		ageNode:  c.ageList.PushFront(k),
		loadTime: time.Now(),
		size:     s,
	}

	c.size += int64(s)

	return true
}

// Add: add a key & entry
func (c *Cache) Add(k interface{}, v interface{}) bool {
	return c.AddWithSize(k, v, 0)
}

// Remove: remove entry by key, return removed entry's value
func (c *Cache) Remove(k interface{}) interface{} {
	c.Lock()
	defer c.Unlock()
	return c.unsafeRemove(k)
}

func (c *Cache) unsafeRemove(k interface{}) interface{} {
	if v, ok := c.m[k]; ok {
		delete(c.m, k)
		c.size -= int64(v.size)
		c.lruList.Remove(v.lruNode)
		c.ageList.Remove(v.ageNode)
		return v.value
	}
	return nil
}

func (c *Cache) removeExpired() {
	// 过期的时刻
	timeLine := time.Now().Add(-c.ttl)
	for elm := c.ageList.Back(); elm != nil; elm = c.ageList.Back() {
		if v, ok := c.m[elm.Value]; ok {
			// 遇到第一个没有过期的，中止
			if v.loadTime.After(timeLine) {
				break
			}
			// 移除过期元素
			c.unsafeRemove(elm.Value)
		}
	}
}

// cull to keep available space over 10% capacity
func (c *Cache) cull() {
	// 超过97%清除过期元素
	waterLine := c.capacity * 97 / 100
	if len(c.m) < waterLine {
		return
	}
	c.removeExpired()

	// 继续清除最早访问的元素，直到有10%空间
	waterLine = c.capacity * 90 / 100
	for len(c.m) > waterLine {
		if elm := c.lruList.Back(); elm != nil {
			c.unsafeRemove(elm.Value)
		}
	}
}
