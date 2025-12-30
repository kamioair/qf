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
	"moduleA"
	"unsafe"
)

func main() {
	// 插件模式的 main 函数不需要执行任何操作
}

//export Init
func Init(settingJson *C.char, settingLen C.int, onWriteCallback C.OnWriteCallback, onReadCallbackPtr uintptr) {
	// 解析qf传入的自定义参数
	setting := map[string]string{}
	err := json.Unmarshal(C.GoBytes(unsafe.Pointer(settingJson), settingLen), &setting)
	if err != nil {
		setting = map[string]string{}
	}
	name := moduleA.Name
	if n, ok := setting["name"]; ok {
		name = n
	}
	
	// 创建配置和服务
	cfg := moduleA.NewConfig()
	serv := moduleA.NewService(cfg)

	// 直接使用 qf.NewPlugin，传入 C 回调函数指针
	qf.NewPlugin(
		name,
		moduleA.Desc,
		moduleA.Version,
		serv,
		cfg,
		uintptr(unsafe.Pointer(onWriteCallback)),
		onReadCallbackPtr,
	)
}
