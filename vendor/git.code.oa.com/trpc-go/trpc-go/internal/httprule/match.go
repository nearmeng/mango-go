package httprule

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/internal/stack"
)

var (
	errNotMatch       = errors.New("not match to the path template")
	errVerbMismatched = errors.New("verb mismatched")
)

// matcher 用于从 Template 中 match 到变量值
type matcher struct {
	components []string          // urlPath: "/foo/bar/baz" => []string{"foo","bar","baz"}
	pos        int               // 当前 match 到的位置，每次 match 都会初始化
	captured   map[string]string // 捕捉到的变量值
	st         *stack.Stack      // 栈，用于辅助 match
}

// matcher 池
var matcherPool = sync.Pool{
	New: func() interface{} {
		return &matcher{}
	},
}

// 把 matcher 放回池中
func putBackMatcher(m *matcher) {
	m.components = nil
	m.pos = 0
	m.captured = nil
	m.st.Reset()
	stackPool.Put(m.st)
	m.st = nil
	matcherPool.Put(m)
}

// stack 池
var stackPool = sync.Pool{
	New: func() interface{} {
		return stack.New()
	},
}

// handle 实现 segment
func (wildcard) handle(m *matcher) error {
	// 防止越界
	if m.pos >= len(m.components) {
		return errNotMatch
	}

	// "*" match 任意一个 component，可以直接 push 进栈
	m.st.Push(m.components[m.pos])
	m.pos++

	return nil
}

// handle 实现 segment
func (deepWildcard) handle(m *matcher) error {
	// 防止越界
	// "**" match 任意个 component，所以 pos 可以等于 len(m.components)
	if m.pos > len(m.components) {
		return errNotMatch
	}

	// 把从 pos 开始到最后一个 component 进行 concatenate 后 push 进栈
	// 字符串拼接最好使用 bytes.Buffer
	var concat bytes.Buffer
	for i := len(m.components) - 1; i >= m.pos; i-- {
		concat.WriteString(m.components[i])
		if i != m.pos {
			concat.WriteString("/")
		}
	}
	m.st.Push(concat.String())
	// "**" 必须是最后一个 segment，pos 直接置为最末
	m.pos = len(m.components)

	return nil
}

// handle 实现 segment
func (l literal) handle(m *matcher) error {
	// 防止越界
	if m.pos >= len(m.components) {
		return errNotMatch
	}

	// 常量必须值等于当前 component
	if m.components[m.pos] != l.String() {
		return errNotMatch
	}

	// match 成功后才 push 进栈
	m.st.Push(m.components[m.pos])
	m.pos++

	return nil
}

// handle 实现 segment
func (v variable) handle(m *matcher) error {
	// 递归 match 变量中的 segments
	for _, segment := range v.segments {
		if err := segment.handle(m); err != nil {
			return err
		}
	}

	// 把 pop 出来的 component 进行 concatenate 后就是 capture 到的 v.fieldPath 对应的值
	concat := make([]string, len(v.segments))
	for i := len(v.segments) - 1; i >= 0; i-- {
		s, ok := m.st.Pop().(string)
		if !ok {
			return errNotMatch
		}
		concat[i] = s
	}
	m.captured[strings.Join(v.fp, ".")] = strings.Join(concat, "/")

	return nil
}

// Match 根据到来的 http 请求 url path 中，match 出变量的值
func (tpl *PathTemplate) Match(urlPath string) (map[string]string, error) {
	// 必须以 '/' 开始
	if !strings.HasPrefix(urlPath, "/") {
		return nil, errNotMatch
	}

	urlPath = urlPath[1:]
	components := strings.Split(urlPath, "/")

	// verb match
	if tpl.verb != "" {
		if !strings.HasSuffix(components[len(components)-1], ":"+tpl.verb) {
			return nil, errVerbMismatched
		}
		idx := len(components[len(components)-1]) - len(tpl.verb) - 1
		if idx <= 0 {
			return nil, errVerbMismatched
		}
		components[len(components)-1] = components[len(components)-1][:idx]
	}

	// 初始化 matcher，match 是高频操作，使用 sync.Pool 内存复用提升性能
	m := matcherPool.Get().(*matcher)
	defer putBackMatcher(m)
	m.components = components
	m.captured = make(map[string]string)
	// Stack 使用 sync.Pool 内存复用提升性能
	m.st = stackPool.Get().(*stack.Stack)

	// segments match
	for _, segment := range tpl.segments {
		if err := segment.handle(m); err != nil {
			return nil, err
		}
	}

	// 检查 pos 是否到达最后
	if m.pos != len(m.components) {
		return nil, errNotMatch
	}

	return m.captured, nil
}
