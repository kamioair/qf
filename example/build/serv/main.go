package main

import (
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
)

func main() {
	// 创建配置和服务
	serv := example.NewService()

	// 启动模块
	module := qf.NewModule(serv)
	module.Run()
}
