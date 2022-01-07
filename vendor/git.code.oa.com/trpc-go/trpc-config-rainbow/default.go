package rainbow

import (
	"errors"
	"strings"
)

var (
	// ErrUnsupportedOperation 不支持的操作
	ErrUnsupportedOperation = errors.New("trpc-config-rainbow: unsupported operation")
	// ErrConfigNotExist 配置不存在
	ErrConfigNotExist = errors.New("trpc-config-rainbow: config not exist")
)

const (
	pluginType = "config"
	pluginName = "rainbow"
)

const (
	RainbowTypeKV    = "kv"
	RainbowTypeTable = "table"
	RainbowTypeGroup = "group"
)

const (
	Sha1   = "sha1"
	Sha256 = "sha256"
)

// spliceString 将 []string 拼接为 []struct
func spliceString(rows []string) string {
	var val strings.Builder
	val.WriteString("[")
	val.WriteString(strings.Join(rows, ","))
	val.WriteString("]")
	return val.String()
}
