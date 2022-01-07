package version

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"
)

// Version ver
var (
	// Version 版本号
	Version    = "go_sdk_v0.3.15" // 增加心跳
	Header     = "rainbow_sdk_version"
	TerminalID = terminalID()
)

func terminalID() string {
	result, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	return base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%d|%d|%d", result.Int64(), time.Now().Unix(), os.Getpid())))
}
