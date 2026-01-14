package main

/*
#include <stdint.h>

// OnWrite函数类型定义
typedef int (*OnWriteCallback)(char*, int, char**);

*/
import "C"
import (
	"encoding/json"
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	"unsafe"
)

func main() {
	// 插件模式的 main 函数不需要执行任何操作
}

//export Init
func Init(settingJson *C.char, settingLen C.int, onWriteCallback C.OnWriteCallback, onReadCallbackPtr uintptr) {
	// 解析qf传入的自定义参数
	setting := map[string]any{}
	err := json.Unmarshal(C.GoBytes(unsafe.Pointer(settingJson), settingLen), &setting)
	if err != nil {
		setting = map[string]any{}
	}

	// 创建配置和服务
	serv := example.NewService(setting)

	// 启动插件
	module := qf.NewPlugin(serv, uintptr(unsafe.Pointer(onWriteCallback)), onReadCallbackPtr)
	module.Run()
}
