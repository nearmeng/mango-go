package main

import (
	"fmt"

	_ "github.com/nearmeng/mango-go/example/statelesssvr/module"
	"github.com/nearmeng/mango-go/server_base/app"
)

func main() {

	server := app.NewServerApp("stateless_svr")

	err := server.Init()
	if err != nil {
		panic(err)
	}

	server.Mainloop()

	err = server.Fini()
	if err != nil {
		fmt.Printf("server fini failed")
	}
}
