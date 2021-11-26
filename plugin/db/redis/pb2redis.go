package redis

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nearmeng/mango-go/plugin/db/pbsupport"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	_metaMap = map[protoreflect.FullName]*DBProtoMeta{}
	_locker  = sync.RWMutex{}
)

// DBProtoMeta db proto meta structure.
type DBProtoMeta struct {
	BuildKey     func(protoreflect.Message) string
	fdNameToFd   map[string]protoreflect.FieldDescriptor
	fdNumToName  map[protowire.Number]string
	IncreaseAble func([]string) bool
}

// BuildKey 后续可用生成函数替代.理论上性能更好.
func BuildKey(pbmsg proto.Message) string {
	rf := pbmsg.ProtoReflect()
	meta := GetDBProtoMeta(rf.Descriptor())
	return meta.BuildKey(rf)
}

//nolint TODO need optimize.
func GetDBProtoMeta(desc protoreflect.Descriptor) *DBProtoMeta {
	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil
	}
	fullName := msgDesc.FullName()
	_locker.RLock()
	meta, ok := _metaMap[fullName]
	_locker.RUnlock()
	if !ok {
		_locker.Lock()
		defer _locker.Unlock()
		meta, ok = _metaMap[fullName]
		if !ok {
			keyNames := pbsupport.FindPrimaryKey(msgDesc)
			keyFds := pbsupport.FindFds(msgDesc, keyNames)
			prefix := string(msgDesc.Name())
			fa := []string{}
			for i := 0; i < len(keyFds); i++ {
				fa = append(fa, "%s")
			}
			fmtStr := fmt.Sprintf("%s:{%s}", prefix, strings.Join(fa, "-"))
			meta = &DBProtoMeta{
				fdNameToFd:  map[string]protoreflect.FieldDescriptor{},
				fdNumToName: map[protowire.Number]string{},
			}

			if len(keyFds) == 1 {
				meta.BuildKey = func(rf protoreflect.Message) string {
					return fmt.Sprintf(fmtStr, rf.Get(keyFds[0]).String())
				}
			} else if len(keyFds) > 1 {
				meta.BuildKey = func(rf protoreflect.Message) string {
					var s []interface{}
					for _, fd := range keyFds {
						s = append(s, rf.Get(fd).String())
					}
					return fmt.Sprintf(fmtStr, s...)
				}
			}

			fields := msgDesc.Fields()
			for i := 0; i < fields.Len(); i++ {
				fd := fields.Get(i)
				meta.fdNameToFd[string(fd.Name())] = fd
				meta.fdNumToName[fd.Number()] = string(fd.Name())
			}
			fdNameToFd := meta.fdNameToFd

			increaseableKeys := pbsupport.BuildIncreaseableFieldsMap(msgDesc)
			meta.IncreaseAble = func(fields []string) bool {
				if len(fields) == 0 {
					return false
				}
				for _, fs := range fields {
					fd, exist := fdNameToFd[fs]
					if !exist {
						return false
					}
					if _, exist := increaseableKeys[string(fd.Name())]; !exist {
						return false
					}
				}
				return true
			}

			_metaMap[fullName] = meta
		}
	}
	return meta
}
