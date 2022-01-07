package types

import (
	"github.com/pkg/errors"
)

// https://iwiki.woa.com/pages/viewpage.action?pageId=46362618

var (
	// ErrSucc success
	ErrSucc = errors.Errorf("%s", "succ")
	// ErrUnauthorized  权限不允许
	ErrUnauthorized = errors.Errorf("%s", "unauthorized")
	// ErrInvalidArg invalid argument
	ErrInvalidArg = errors.Errorf("%s", "invalid argument")
	// ErrConfigServer config server internal error
	ErrConfigServer = errors.Errorf("%s", "config server internal error")
	// ErrRPC rpc request error connect error ...
	ErrRPC = errors.Errorf("%s", "rpc request error")
	// ErrRPCTimeout rpc timeout
	ErrRPCTimeout = errors.Errorf("%s", "rpc timeout")
	// ErrVerNoChange  version no change
	ErrVerNoChange = errors.Errorf("%s", "version no change")
	// ErrNoConfig no configuration
	ErrNoConfig = errors.Errorf("%s", "no configuration")
	// ErrNoAnyVersion  no any version
	ErrNoAnyVersion = errors.Errorf("%s", "no any version")
	// ErrPollingAgain need polling again
	ErrPollingAgain = errors.Errorf("%s", "pollin request again")
	// ErrUnknown unkonown error
	ErrUnknown = errors.Errorf("%s", "unkonown")
	// ErrExistWatch exist watch
	ErrExistWatch = errors.Errorf("%s", "exist watch")
	// ErrNotFoundKey 客户端指定获取某个key，但这个key不存在
	ErrNotFoundKey = errors.Errorf("not found key")
)
var errTable = map[int32]error{
	0:   ErrSucc,
	100: ErrUnauthorized,
	200: ErrInvalidArg,
	500: ErrConfigServer,
	600: ErrRPC,
	601: ErrRPCTimeout,
	702: ErrVerNoChange,
	704: ErrNoConfig,
	707: ErrNoAnyVersion,
	706: ErrNotFoundKey,
	708: ErrPollingAgain,
}

// Code2Error 根据查error
func Code2Error(code int32) error {
	if err, ok := errTable[code]; ok {
		return err
	}
	return ErrUnknown
}
