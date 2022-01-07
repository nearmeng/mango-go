# GO_REUSEPORT
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-750d233f6e054a51ad8f78cf154ee93c/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-750d233f6e054a51ad8f78cf154ee93c/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject) [![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-750d233f6e054a51ad8f78cf154ee93c)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-750d233f6e054a51ad8f78cf154ee93c) [![GoDoc](https://img.shields.io/badge/API%20Docs-GoDoc-green)](http://godoc.oa.com/git.code.oa.com/trpc-go/go_reuseport)

[![Build Status](https://travis-ci.org/kavu/go_reuseport.png?branch=master)](https://travis-ci.org/kavu/go_reuseport)
[![codecov](https://codecov.io/gh/kavu/go_reuseport/branch/master/graph/badge.svg)](https://codecov.io/gh/kavu/go_reuseport)
[![GoDoc](https://godoc.org/github.com/kavu/go_reuseport?status.png)](https://godoc.org/github.com/kavu/go_reuseport)

**GO_REUSEPORT** is a little expirement to create a `net.Listener` that supports [SO_REUSEPORT](http://lwn.net/Articles/542629/) socket option.

For now, Darwin and Linux (from 3.9) systems are supported. I'll be pleased if you'll test other systems and tell me the results.
 documentation on [godoc.org](http://godoc.org/github.com/kavu/go_reuseport "go_reuseport documentation").

## Example ##

```go
package main

import (
  "fmt"
  "html"
  "net/http"
  "os"
  "runtime"
  "github.com/kavu/go_reuseport"
)

func main() {
  listener, err := reuseport.Listen("tcp", "localhost:8881")
  if err != nil {
    panic(err)
  }
  defer listener.Close()

  server := &http.Server{}
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Println(os.Getgid())
    fmt.Fprintf(w, "Hello, %q\n", html.EscapeString(r.URL.Path))
  })

  panic(server.Serve(listener))
}
```

Now you can run several instances of this tiny server without `Address already in use` errors.

## Thanks

Inspired by [Artur Siekielski](https://github.com/aartur) [post](http://freeprogrammersblog.vhex.net/post/linux-39-introdued-new-way-of-writing-socket-servers/2) about `SO_REUSEPORT`.

