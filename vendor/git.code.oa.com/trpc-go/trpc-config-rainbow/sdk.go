package rainbow

import (
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/v3/keep"
	"git.code.oa.com/rainbow/golang-sdk/v3/watch"
	pb "git.code.oa.com/rainbow/proto/api/configv3"
)

// SDK rainbow sdk接口
type SDK interface {
	AddWatcher(w watch.Watcher, opts ...types.AssignGetOption) error
	Get(key string, opts ...types.AssignGetOption) (string, error)
	GetTable(opts ...types.AssignGetOption) (*pb.TableList, error)
	GetGroup(opts ...types.AssignGetOption) (keep.Group, error)
}
