// Package stack 提供一个非线程安全栈
package stack

// Stack 非线程安全栈
type Stack struct {
	top  *node
	size int
}

type node struct {
	value interface{}
	prev  *node
}

// New 创建一个栈
func New() *Stack {
	return &Stack{}
}

// Size 返回栈长度
func (st *Stack) Size() int {
	return st.size
}

// Reset 重置栈
func (st *Stack) Reset() {
	st.top = nil
	st.size = 0
}

// Push push 一个元素进栈
func (st *Stack) Push(value interface{}) {
	newNode := &node{
		value: value,
		prev:  st.top,
	}
	st.top = newNode
	st.size++
}

// Pop pop 一个元素出栈
func (st *Stack) Pop() interface{} {
	if st.size == 0 {
		return nil
	}
	topNode := st.top
	st.top = topNode.prev
	topNode.prev = nil
	st.size--
	return topNode.value
}

// Peek 查看栈顶元素
func (st *Stack) Peek() interface{} {
	if st.size == 0 {
		return nil
	}
	return st.top.value
}
