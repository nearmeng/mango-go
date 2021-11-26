// Package mysql db mysql support plugin.
package mysql

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nearmeng/mango-go/plugin/db/pbsupport"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	_CreateTableTemplate string = `CREATE TABLE IF NOT EXISTS {{.TableName}} (
	{{range .CreateSQLs}}
		{{.}} ,
	{{end}}
	PRIMARY KEY ( {{.PrimaryKeys}} )
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='{{.DescFullName}}'`
)

var (
	_BlobKindCreateSQLFmt = "`%s` blob"
	_FdKindToSQLCreate    = map[protoreflect.Kind]string{
		protoreflect.BoolKind:     "`%s` tinyint NOT NULL DEFAULT 0",
		protoreflect.EnumKind:     "`%s` int NOT NULL DEFAULT 0",
		protoreflect.Int32Kind:    "`%s` int NOT NULL DEFAULT 0",
		protoreflect.Sint32Kind:   "`%s` int NOT NULL DEFAULT 0",
		protoreflect.Uint32Kind:   "`%s` int UNSIGNED NOT NULL DEFAULT 0",
		protoreflect.Int64Kind:    "`%s` bigint NOT NULL DEFAULT 0",
		protoreflect.Sint64Kind:   "`%s` bigint NOT NULL DEFAULT 0",
		protoreflect.Uint64Kind:   "`%s` bigint UNSIGNED NOT NULL DEFAULT 0",
		protoreflect.Sfixed32Kind: "`%s` int NOT NULL DEFAULT 0",
		protoreflect.Fixed32Kind:  "`%s` int NOT NULL DEFAULT 0",
		protoreflect.FloatKind:    "`%s` float NOT NULL DEFAULT 0",
		protoreflect.Sfixed64Kind: "`%s` bigint NOT NULL DEFAULT 0",
		protoreflect.DoubleKind:   "`%s` double NOT NULL DEFAULT 0",
		protoreflect.StringKind:   "`%s` text",
		protoreflect.BytesKind:    _BlobKindCreateSQLFmt,
		protoreflect.MessageKind:  _BlobKindCreateSQLFmt,
		protoreflect.GroupKind:    _BlobKindCreateSQLFmt,
	}
	// TODO: if string as a primkey , must use varchar and set max size.
	_StringKindPrimaryKey = "`%s` varchar(255) NOT NULL"
)

var (
	_metaMap = map[protoreflect.FullName]*DBProtoMeta{}
	_locker  = sync.RWMutex{}
)

// ColumnInfo sql column info.
type ColumnInfo struct {
	CreateSQL string
}

// TableInfo sql table info.
type TableInfo struct {
	TableName    string
	DescFullName string
	PrimaryKeys  string
	CreateSQLs   []string
}

func newTableInfo(msgDesc protoreflect.MessageDescriptor) (*TableInfo, error) {
	primkeys := pbsupport.FindPrimaryKey(msgDesc)
	if len(primkeys) == 0 {
		return nil, fmt.Errorf("no primkey msg=%s", msgDesc.FullName())
	}
	ti := &TableInfo{
		TableName:    string(msgDesc.Name()),
		DescFullName: string(msgDesc.FullName()),
		PrimaryKeys:  strings.Join(primkeys, ","),
	}
	fields := msgDesc.Fields()
	ti.CreateSQLs = make([]string, fields.Len())
	keysMap := pbsupport.BuildKeyFieldsMap(msgDesc)
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if pbsupport.IsFdMarshalToBlob(fd) {
			ti.CreateSQLs[i] = fmt.Sprintf(_BlobKindCreateSQLFmt, fd.Name())
		} else {
			_, exist := keysMap[string(fd.Name())]
			if fd.Kind() == protoreflect.StringKind && exist {
				ti.CreateSQLs[i] = fmt.Sprintf(_StringKindPrimaryKey, fd.Name())
			} else {
				ti.CreateSQLs[i] = fmt.Sprintf(_FdKindToSQLCreate[fd.Kind()], fd.Name())
			}
		}
	}
	return ti, nil
}

type sqlPkg struct {
	str    string
	params []interface{}
}

// DBProtoMeta db proto meta structure.
type DBProtoMeta struct {
	SelectFieldsPkg func(protoreflect.Message, []string) *sqlPkg
	ReplacePkg      func(map[string]interface{}) *sqlPkg
	InsertPkg       func(map[string]interface{}) *sqlPkg
	UpdatePkg       func(protoreflect.Message, map[string]interface{}) *sqlPkg
	IncreasePkg     func(protoreflect.Message, []string) *sqlPkg
	DeleteSQLPkg    func(protoreflect.Message) *sqlPkg
}

//nolint get db prot meta info.
func GetDBProtoMeta(desc protoreflect.Descriptor) *DBProtoMeta {
	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil
	}
	msgfullName := msgDesc.FullName()
	_locker.RLock()
	msgmeta, ok := _metaMap[msgfullName]
	_locker.RUnlock()
	if !ok {
		_locker.Lock()
		defer _locker.Unlock()
		msgmeta, ok = _metaMap[msgfullName]
		if !ok {
			keyNames := pbsupport.FindPrimaryKey(msgDesc)
			keyFds := pbsupport.FindFds(msgDesc, keyNames)
			keysMap := pbsupport.BuildKeyFieldsMap(msgDesc)
			kss := make([]string, len(keyNames))
			for i, k := range keyNames {
				kss[i] = fmt.Sprintf("%s=?", k)
			}
			ks := strings.Join(kss, ",")
			fdsNum := msgDesc.Fields().Len()

			msgmeta = &DBProtoMeta{}

			selectAllSQLFmt := fmt.Sprintf("select * from %s where %s", desc.Name(), ks)
			selectFieldsSQLFmt := fmt.Sprintf("select %%s from %s where %s", desc.Name(), ks)
			msgmeta.SelectFieldsPkg = func(rf protoreflect.Message, fields []string) *sqlPkg {
				params := make([]interface{}, len(keyFds))
				for i, fd := range keyFds {
					params[i] = rf.Get(fd).String()
				}
				if len(fields) > 0 {
					return &sqlPkg{
						str:    fmt.Sprintf(selectFieldsSQLFmt, strings.Join(fields, ",")),
						params: params,
					}
				}
				return &sqlPkg{
					str:    selectAllSQLFmt,
					params: params,
				}
			}

			replaceSqlFmt := fmt.Sprintf("replace into %s set ", desc.Name())
			msgmeta.ReplacePkg = func(m map[string]interface{}) *sqlPkg {
				cnt := len(m)
				if cnt == 0 {
					return nil
				}
				params := make([]interface{}, cnt)
				keys := make([]string, cnt)
				i := 0
				for k, v := range m {
					params[i] = v
					keys[i] = fmt.Sprintf("%s=?", k)
					i++
				}
				return &sqlPkg{
					str:    replaceSqlFmt + strings.Join(keys, ","),
					params: params,
				}
			}

			updateSQLFmt := fmt.Sprintf("update %s set %%s where %s", desc.Name(), ks)
			msgmeta.UpdatePkg = func(rf protoreflect.Message, m map[string]interface{}) *sqlPkg {
				cnt := len(m)
				if cnt == 0 {
					return nil
				}
				params := make([]interface{}, 0, cnt)
				keys := make([]string, 0, cnt)
				for k, v := range m {
					if _, exist := keysMap[k]; !exist {
						params = append(params, v)
						keys = append(keys, fmt.Sprintf("%s=?", k))
					}
				}

				for _, fd := range keyFds {
					params = append(params, rf.Get(fd).String())
				}

				return &sqlPkg{
					str:    fmt.Sprintf(updateSQLFmt, strings.Join(keys, ",")),
					params: params,
				}
			}

			increaseableKeys := pbsupport.BuildIncreaseableFieldsMap(msgDesc)
			increaseSQLFmt := fmt.Sprintf("update %s set %%s where %s", desc.Name(), ks)
			msgmeta.IncreasePkg = func(rf protoreflect.Message, fields []string) *sqlPkg {
				if len(fields) == 0 {
					return nil
				}
				params := make([]interface{}, 0, fdsNum)
				keys := make([]string, 0, len(fields))
				for _, k := range fields {
					if _, exist := increaseableKeys[k]; !exist {
						return nil
					}
					keys = append(keys, fmt.Sprintf("%s=%s+1", k, k))
				}

				for _, fd := range keyFds {
					params = append(params, rf.Get(fd).String())
				}

				return &sqlPkg{
					str:    fmt.Sprintf(increaseSQLFmt, strings.Join(keys, ",")),
					params: params,
				}
			}

			insertSqlFmt := fmt.Sprintf("insert into %s (%%s) values (%%s) ", desc.Name())
			msgmeta.InsertPkg = func(m map[string]interface{}) *sqlPkg {
				cnt := len(m)
				if cnt == 0 {
					return nil
				}
				params := make([]interface{}, cnt)
				keys := make([]string, cnt)
				i := 0
				for k, v := range m {
					params[i] = v
					keys[i] = k
					i++
				}
				ph := strings.Repeat("?,", cnt-1) + "?"
				return &sqlPkg{
					str:    fmt.Sprintf(insertSqlFmt, strings.Join(keys, ","), ph),
					params: params,
				}
			}

			deleteSQLFmt := fmt.Sprintf("delete from %s where %s", desc.Name(), ks)
			msgmeta.DeleteSQLPkg = func(rf protoreflect.Message) *sqlPkg {
				params := make([]interface{}, len(keyFds))
				for i, fd := range keyFds {
					params[i] = rf.Get(fd).String()
				}
				return &sqlPkg{
					str:    deleteSQLFmt,
					params: params,
				}
			}

			_metaMap[msgfullName] = msgmeta
		}
	}
	return msgmeta
}
