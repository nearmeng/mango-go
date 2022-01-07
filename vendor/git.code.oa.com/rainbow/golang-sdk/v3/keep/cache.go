package keep

import (
	"sync"

	v3 "git.code.oa.com/rainbow/proto/api/configv3"
	"github.com/qdm12/reprint"
)

// Group 存储一个group数据
type Group map[string]interface{}

// GroupWrapper group 包装一层
type GroupWrapper struct {
	Value       Group         `json:"value,omitempty"` // kv
	Version     string        `json:"version,omitempty"`
	Mutex       sync.RWMutex  `json:"-"`
	VersionID   uint64        `json:"versionId,omitempty"`
	VersionName string        `json:"versionName,omitempty"`
	StructType  int32         `json:"struct_type,omitempty"`
	TBL         *v3.TableList `json:"tbl,omitempty"`
	// KVL         *v3.KVList       `json:"kvl",omitempty"`
	FDL  *v3.FileDataList `json:"fdl,omitempty"`
	FDML FileMetaList     `json:"-"`
}

// UpdateType 更新类型
type UpdateType int32

const (
	// UpdateTypeGetKey get单key触发更新
	UpdateTypeGetKey UpdateType = 0
	// UpdateTypeGetGroup get group 触发更新
	UpdateTypeGetGroup UpdateType = 1
	// UpdateTypePollKey poll单key触发update
	UpdateTypePollKey UpdateType = 2
	// UpdateTypePollGroup poll group
	UpdateTypePollGroup UpdateType = 3
)

// CopyKeyValueItem  KeyValueItem
// https://www.reddit.com/r/golang/comments/e2pd8y/deep_copying_done_right/
func CopyKeyValueItem(src *v3.Item) *v3.Item {
	dst := &v3.Item{}
	reprint.FromTo(src, dst)
	return dst
}

// CopyGroupWithTable copy group
func CopyGroupWithTable(src *GroupWrapper) GroupWrapper {
	dst := &GroupWrapper{}
	reprint.FromTo(src, dst)
	return *dst
}

// FileMeta file meta data
type FileMeta struct {
	FileVersion  string `json:"file_version,omitempty"`
	IsEditable   bool   `json:"is_editable,omitempty"`
	IsGBK        bool   `json:"is_gbk,omitempty"`
	MD5          string `json:"md5,omitempty"`
	MTime        string `json:"mtime,omitempty"`
	Name         string `json:"name,omitempty"`
	Size         int    `json:"size,omitempty"`
	URL          string `json:"url,omitempty"`
	Content      string `json:"content,omitempty"`
	SecrectKey   string `json:"secret_key,omitempty"`
	SecretKeyMD5 string `json:"secret_key_md5,omitempty"`
}

// FileMetaList file meta data list
type FileMetaList []*FileMeta
