package rainbow

import (
	runtime "git.code.oa.com/trpc-go/trpc-metrics-runtime"
)

// Version rainbow version
const Version = "v0.1.19"

func init() {
	go runtime.StatReport(Version, "rainbow")
}
