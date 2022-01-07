package rpc

type server interface {
	Init() error
	Mainloop()
	UnInit() error
}
