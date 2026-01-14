package main

import (
	"encoding/json"
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	"os"
)

func main() {
	// 获取qf传入的自定义参数
	setting := map[string]any{}
	if len(os.Args) > 1 {
		_ = json.Unmarshal([]byte(os.Args[1]), &setting)
	}

	// 创建配置和服务
	serv := example.NewService(setting)

	// 启动模块
	module := qf.NewModule(serv)
	module.Run()
}
