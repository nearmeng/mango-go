package cache

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func (c *Cache) addTestItem(i int) bool {
	return c.Add(
		i,
		fmt.Sprintf("string #%d", i),
	)
}

func TestAddWithSize(t *testing.T) {
	c := New()
	c.AddWithSize(1, "1", 1)
	if c.Size() != 1 {
		t.Errorf("fail: size")
	}
	c.AddWithSize(2, "2", 2)
	if c.Size() != 3 {
		t.Errorf("fail: size")
	}
}

func TestAddRemove(t *testing.T) {
	c := New()

	if !c.addTestItem(1) || c.Get(1) == nil {
		t.Errorf("fail: add entry 1")
	}

	if !c.addTestItem(2) || c.Get(2) == nil {
		t.Errorf("fail: add entry 2")
	}

	if c.addTestItem(1) {
		t.Errorf("fail: add dup entry")
	}

	item := c.Remove(1)

	if item == nil || c.Get(1) != nil {
		t.Errorf("fail: remove entry")
	}
}

func TestHitAndMiss(t *testing.T) {
	c := New()

	c.addTestItem(1)
	fmt.Println(c.Get(1))

	c.addTestItem(2)
	fmt.Println(c.Get(2))

	if c.Hits() != 2 {
		t.Errorf("fail: Hits expect 2, but got %d", c.Hits())
	}

	c.Get(3)
	if c.Misses() != 1 {
		t.Errorf("fail: Misses expect 2, but got %d", c.Misses())
	}
}

func TestCahce_Update(t *testing.T) {
	c := New()
	c.addTestItem(1)

	c.Update(1, "new")

	v := c.Get(1)
	if v == nil || v != "new" {
		t.Errorf("update failed")
	}

	fmt.Printf("value:%s\n", v)
}

func TestExpire(t *testing.T) {
	c := New()
	c.SetCapacity(5)
	c.SetTTL(time.Millisecond)

	// expire
	c.addTestItem(1)
	time.Sleep(time.Millisecond * 5)
	//c.addTestItem(2)
	if c.Get(1) != nil {
		t.Errorf("fail: entry 1 should expired")
	}

	// 4/5 = 80%
	c.addTestItem(1)
	c.addTestItem(2)
	c.addTestItem(3)
	c.addTestItem(4)
	fmt.Printf("size: %d/%d\n", c.Len(), c.capacity)

	time.Sleep(time.Millisecond * 5)

	c.addTestItem(5)
	fmt.Printf("size: %d/%d\n", c.Len(), c.capacity)

	// should cull expired
	if c.Len() != 1 {
		t.Errorf("fail: expired entry did not remove")
	}
	if c.Get(5) == nil {
		t.Errorf("fail: entry 5 did not exist")
	}
	if c.Get(6) != nil {
		t.Errorf("fail: entry 6 should not exist")
	}

	//fmt.Println(c.Get(4))
}

func TestCull(t *testing.T) {
	c := New()
	c.SetCapacity(5)
	c.SetTTL(time.Millisecond)

	// 4/5 = 80%
	c.addTestItem(1)
	c.addTestItem(2)
	c.addTestItem(3)
	c.addTestItem(4)
	c.addTestItem(5)
	fmt.Printf("size: %d/%d\n", c.Len(), c.capacity)

	if !c.addTestItem(6) {
		t.Errorf("fail: add entry 6")
	}
	if c.Get(6) == nil {
		t.Errorf("fail: get entry 6")
	}

	if c.Get(1) != nil {
		t.Errorf("fail: cull")
	}
	if c.Get(2) == nil {
		t.Errorf("fail: cull")
	}
	fmt.Printf("size: %d/%d\n", c.Len(), c.capacity)
}

func loopAdd(c *Cache) {
	i := 0
	for {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		i++
		c.addTestItem(i)
		//if c.addTestItem(i) {
		//	fmt.Printf("#%d added %d: %v\n", id, i, c.Get(i).value)
		//} else {
		//	fmt.Printf("#%d exist %d\n", id, i)
		//}
	}
}

func TestConcurrency(t *testing.T) {
	c := New()

	go loopAdd(c)
	go loopAdd(c)
	go loopAdd(c)

	time.Sleep(2 * time.Second)
	fmt.Printf("size: %d/%d\n", c.Len(), c.capacity)
}
