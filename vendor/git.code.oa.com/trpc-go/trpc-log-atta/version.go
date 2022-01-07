package attalog

import (
	runtime "git.code.oa.com/trpc-go/trpc-metrics-runtime"
)

// Version attalog version
const Version = "v0.1.12"

func init() {
	go runtime.StatReport(Version, "attalog")
}
