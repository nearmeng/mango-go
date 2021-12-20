package res

import (
	"google.golang.org/protobuf/proto"
)

// Table ...
type Table interface {
	Insert(key int32, item proto.Message)
	Find(key int32) proto.Message
	ForEach(f func(cfg proto.Message))
	Count() int
	HashCode() string
}

// TableConfig ...
type TableConfig struct {
	ID        int32                     // 数据表ID
	FileName  string                    // 文件名
	FieldName string                    // Key字段名
	Message   proto.Message             // 表结构
	PostFunc  func(id int32, tbl Table) // 加载后执行
}

// Config ...
type Config struct {
	ResPath string
	Tables  []TableConfig
}

type ResTable struct {
	items    map[int32]proto.Message
	hashCode string
}

func NewResTable() *ResTable {
	return &ResTable{
		items: map[int32]proto.Message{},
	}
}

// Insert ...
func (tbl *ResTable) Insert(key int32, item proto.Message) {
	tbl.items[key] = item
}

// Find ...
func (tbl *ResTable) Find(key int32) proto.Message {
	item, ok := tbl.items[key]
	if !ok {
		return nil
	}
	return item
}

// ForEach ...
func (tbl *ResTable) ForEach(f func(proto.Message)) {
	for _, item := range tbl.items {
		f(item)
	}
}

// Count ...
func (tbl *ResTable) Count() int {
	return len(tbl.items)
}

// HashCode ...
func (tbl *ResTable) HashCode() string {
	return tbl.hashCode
}
