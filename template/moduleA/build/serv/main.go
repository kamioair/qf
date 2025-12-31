package main

import (
	"github.com/kamioair/qf"
	"moduleA"
)

func main() {
	serv := moduleA.NewService(nil)
	module := qf.NewModule(serv)
	module.Run()
}
