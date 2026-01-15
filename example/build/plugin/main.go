package main

/*
#include <stdint.h>

// OnWrite函数类型定义
typedef int (*OnWriteCallback)(char*, int, char**);

*/
import "C"
import (
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	"unsafe"
)

func main() {
	// 插件模式的 main 函数不需要执行任何操作
}

//export Init
func Init(onWriteCallback C.OnWriteCallback, onReadCallbackPtr uintptr) {
	// 创建配置和服务
	serv := example.NewService()

	// 启动插件
	module := qf.NewPlugin(serv, uintptr(unsafe.Pointer(onWriteCallback)), onReadCallbackPtr)
	module.Run()
}
