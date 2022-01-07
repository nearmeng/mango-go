package m007

import (
	runtime "git.code.oa.com/trpc-go/trpc-metrics-runtime"
)

// Version 007 version
const Version = "v0.4.7"

func init() {
	go runtime.StatReport(Version, "m007")
}
