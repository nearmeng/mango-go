package naming

import (
	runtime "git.code.oa.com/trpc-go/trpc-metrics-runtime"
)

// Version polaris version
const Version = "v0.2.7"

func init() {
	go runtime.StatReport(Version, "polaris")
}
