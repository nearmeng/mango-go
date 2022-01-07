// Package dat 提供一个 double array trie
// dat 用于 RESTful API 中，过滤被 HttpRule 引用过的 protobuf fields，
// 使其不会再被 http request query parameters 重复引用
package dat

import (
	"errors"
	"sort"
)

var (
	errByDictOrder = errors.New("not by dict order")
	errEncoded     = errors.New("field name not encoded")
)

const (
	defaultArraySize         = 64   // dat 数组初始大小
	minExpansionRate         = 1.05 // 最小每次扩容比率，经验值
	nextCheckPosStrategyRate = 0.95 // 设置 nextCheckPos 策略，经验值
)

// DoubleArrayTrie 双数组字典树
// 基于 https://github.com/komiya-atsushi/darts-java 实现
// 状态转移方程：
//   base[0] = 1
//   base[s] + c = t
//   check[t] = base[s]
type DoubleArrayTrie struct {
	base         []int      // base 数组
	check        []int      // check 数组
	used         []bool     // used 数组
	size         int        // base/check/used 已使用的大小
	allocSize    int        // base/check/used 数组分配的大小
	fps          fieldPaths // fieldPaths 数据
	dict         fieldDict  // dictCodeOfFieldName 码表
	progress     int        // 处理完的 fieldPath 数量
	nextCheckPos int        // 记录下一次从此开始寻找 begin 值的下标，避免每次从 0 开始
}

// node 双数组字典树节点
type node struct {
	code  int // 约定 code = dictCodeOfFieldName + 1, dictCodeOfFieldName: [0, 1, 2, ..., n-1]
	depth int // 节点在树中的深度
	left  int // 左边界
	right int // 右边界
}

// Build 静态构造双数组字典树
func Build(fps [][]string) (*DoubleArrayTrie, error) {
	// 排序
	sort.Sort(fieldPaths(fps))

	// 初始化 dat
	dat := &DoubleArrayTrie{
		fps:  fps,
		dict: newFieldDict(fps),
	}
	dat.resize(defaultArraySize)
	dat.base[0] = 1

	// 根节点处理
	root := &node{
		right: len(dat.fps),
	}
	children, err := dat.fetch(root)
	if err != nil {
		return nil, err
	}
	_, err = dat.insert(children)
	if err != nil {
		return nil, err
	}

	// 缩容
	dat.resize(dat.size)

	return dat, nil
}

// fetch 给定父节点，返回子节点
// 譬如 dat 中的 fps 如下：
//  ["foobar", "foo", "bar"]
//  ["foobar", "baz"]
//  ["foo", "qux"]
// 则 children, _ := dat.fetch(root)，children 为： ["foobar", "foo"]，depth 都为 1
func (dat *DoubleArrayTrie) fetch(parent *node) ([]*node, error) {
	var children []*node // 要返回的子节点
	var prev int         // 用于记录上一个子节点的 code

	// 搜索范围 [parent.left, parent.right)
	// 对于root 节点，搜索范围 [0, len(dat.fps))
	for i := parent.left; i < parent.right; i++ {
		if len(dat.fps[i]) < parent.depth { // fps[i] 已经被 fetch 完了
			continue
		}

		var curr int // 当前子节点的 code
		if len(dat.fps[i]) > parent.depth {
			v, ok := dat.dict[dat.fps[i][parent.depth]]
			if !ok { // 没记录过，报错
				return nil, errEncoded
			}
			curr = v + 1 // code = dictCodeOfFieldName + 1
		}

		// 非字典序，报错
		if prev > curr {
			return nil, errByDictOrder
		}

		// 正常如果 curr == prev，跳过
		// 但是 curr == prev && len(children) == 0 的情况例外，
		// 说明 fps[i] 正好 fetch 到最后，还要加一个空节点，类似于结束符
		if curr != prev || len(children) == 0 {
			// 当前子节点，不用更新 right，right 让下一个子节点来更新
			child := &node{
				code:  curr,
				depth: parent.depth + 1, // 子节点深度要 +1
				left:  i,
			}
			// 更新上一个子节点 right
			if len(children) != 0 {
				children[len(children)-1].right = i
			}
			children = append(children, child)
		}

		prev = curr
	}

	// 更新最后一个子节点 right
	if len(children) > 0 {
		children[len(children)-1].right = parent.right // 和父节点 right 一样
	}

	return children, nil
}

// maxInt 取最大 int 值
func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// maxFloat 取最大 float64 值
func maxFloat(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}

// loopForBegin 寻找满足条件的 begin 值
func (dat *DoubleArrayTrie) loopForBegin(children []*node) (int, error) {
	begin := 0                                          // 要寻找 begin
	pos := maxInt(children[0].code, dat.nextCheckPos-1) // 避免从下标 0 开始寻找 begin 的值
	numOfNonZero := 0                                   // 遇见的非零数个数
	first := true                                       // 是否是第一次碰见非零数

	if dat.allocSize <= pos { // 扩容
		dat.resize(pos + 1)
	}

	for {
		pos++
		if dat.allocSize <= pos { // 扩容
			dat.resize(pos + 1)
		}
		if dat.check[pos] != 0 { // 被占用
			numOfNonZero++
			continue
		} else {
			if first {
				dat.nextCheckPos = pos
				first = false
			}
		}

		// 尝试此 begin 值
		begin = pos - children[0].code

		// 扩容，直接用最后一个要插入的子节点的 pos 来对比
		if lastChildPos := begin + children[len(children)-1].code; dat.allocSize <= lastChildPos {
			// 计算扩容比例用 fieldPath 总数 / (已处理的 fieldPath 数 + 1) 计算，但不能低于 1.05
			rate := maxFloat(minExpansionRate, float64(1.0*len(dat.fps)/(dat.progress+1)))
			dat.resize(int(float64(dat.allocSize) * rate))
		}

		if dat.used[begin] { // begin 不能重复
			continue
		}

		// 检查其余子节点是否能插入
		conflict := func() bool {
			for i := 1; i < len(children); i++ {
				if dat.check[begin+children[i].code] != 0 {
					return true
				}
			}
			return false
		}
		// 有冲突直接下一个 pos
		if conflict() {
			continue
		}
		// 没有冲突，找到满足条件的 begin
		break
	}

	// 如果 nextCheckPos 和 pos 之间基本上都被占用了，则把 nextCheckPos 置为 pos
	if float64((1.0*numOfNonZero)/(pos-dat.nextCheckPos+1)) >= nextCheckPosStrategyRate {
		dat.nextCheckPos = pos
	}

	return begin, nil
}

// insert 往双数组中插入子节点，返回要寻找的 begin 值
func (dat *DoubleArrayTrie) insert(children []*node) (int, error) {
	// 寻找 begin 值
	begin, err := dat.loopForBegin(children)
	if err != nil {
		return 0, err
	}

	dat.used[begin] = true
	dat.size = maxInt(dat.size, begin+children[len(children)-1].code+1)

	// 赋值 check 数组
	for i := range children {
		dat.check[begin+children[i].code] = begin
	}

	// dfs
	for _, child := range children {
		grandchildren, err := dat.fetch(child)
		if err != nil {
			return 0, err
		}
		if len(grandchildren) == 0 { // 没有子节点了
			dat.base[begin+child.code] = -child.left - 1
			dat.progress++
			continue
		}
		t, err := dat.insert(grandchildren)
		if err != nil {
			return 0, err
		}
		// 赋值 base 数组
		dat.base[begin+child.code] = t
	}

	return begin, nil
}

// resize 改变数组大小
func (dat *DoubleArrayTrie) resize(newSize int) {
	newBase := make([]int, newSize, newSize)
	newCheck := make([]int, newSize, newSize)
	newUsed := make([]bool, newSize, newSize)

	if dat.allocSize > 0 {
		copy(newBase, dat.base)
		copy(newCheck, dat.check)
		copy(newUsed, dat.used)
	}

	dat.base = newBase
	dat.check = newCheck
	dat.used = newUsed

	dat.allocSize = newSize
}

// CommonPrefixSearch 判断输入 fieldPath 是否和 dat 中的 fps 有共同的前缀
func (dat *DoubleArrayTrie) CommonPrefixSearch(fieldPath []string) bool {
	pos := 0
	baseValue := dat.base[0]

	for _, name := range fieldPath {
		// 获取 dict code
		v, ok := dat.dict[name]
		if !ok {
			break
		}
		code := v + 1 // code = dictCodeOfFieldName + 1

		// 判断是否到叶节点，即判断状态转移到下个节点是 NULL 节点
		if baseValue == dat.check[baseValue] && dat.base[baseValue] < 0 { // 已到叶节点，是前缀
			return true
		}

		// 状态转移
		pos = baseValue + code
		if pos >= len(dat.check) || baseValue != dat.check[pos] { // check 对不上
			return false
		}
		baseValue = dat.base[pos]
	}

	// 为最后一次状态转移再判断一次是否到叶节点
	if baseValue == dat.check[baseValue] && dat.base[baseValue] < 0 { // 已到叶节点，是前缀
		return true
	}

	return false
}

type fieldPaths [][]string

// Len 实现 sort.Interface
func (fps fieldPaths) Len() int { return len(fps) }

// Swap 实现 sort.Interface
func (fps fieldPaths) Swap(i, j int) { fps[i], fps[j] = fps[j], fps[i] }

// Less 实现 sort.Interface
func (fps fieldPaths) Less(i, j int) bool {
	var k int
	for k = 0; k < len(fps[i]) && k < len(fps[j]); k++ {
		if fps[i][k] < fps[j][k] {
			return true
		}
		if fps[i][k] > fps[j][k] {
			return false
		}
	}
	return k < len(fps[j])
}

type fieldDict map[string]int // FieldName -> DictCodeOfFieldName

func newFieldDict(fps fieldPaths) fieldDict {
	fields := make([]string, 0)
	dict := make(map[string]int)

	// 去重
	for _, fieldPath := range fps {
		for _, name := range fieldPath {
			dict[name] = 0
		}
	}

	// 字典序排序
	for name := range dict {
		fields = append(fields, name)
	}
	sort.Sort(sort.StringSlice(fields))

	// 码表赋值
	code := 0
	for _, name := range fields {
		dict[name] = code
		code++
	}
	return dict
}
