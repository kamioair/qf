package main

/*
#include <string.h>
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>

// OnWrite函数
typedef int (*OnWriteCallback)(char*, int, char**);
static int OnWriteHandler(OnWriteCallback cb, char* respBytes, int respLen, char** outErrorMsg) {
	int result = cb(respBytes, respLen, outErrorMsg);
	return result;
}

// OnRead函数类型定义
typedef void (*OnReadCallback)(char*, int);

*/
import "C"
import (
	"fmt"
	"github.com/kamioair/qf"
	"moduleA"
	"syscall"
	"unsafe"
)

func main() {

}

//export Init
func Init(onWriteCallback C.OnWriteCallback, onReadCallbackPtr uintptr) {
	cfg := moduleA.NewConfig()
	serv := moduleA.NewService(cfg)

	// 创建写入函数,通过函数指针调用
	onWrite := func(data []byte) error {
		size := C.int(len(data))
		packStr := (*C.char)(C.CBytes(data))

		// 创建错误信息指针变量，用于接收回调返回的错误
		var errorMsg *C.char

		// 调用回调函数，将 errorMsg 的地址传递给 C 函数
		// C 函数会把这个地址传递给 launcher 的回调，回调会设置这个指针的值
		result := C.OnWriteHandler(onWriteCallback, packStr, size, &errorMsg)

		// 释放通过 C.CBytes 分配的内存
		C.free(unsafe.Pointer(packStr))

		// 检查返回结果
		if result != 0 {
			// 如果回调函数返回错误码，检查是否有错误信息
			if errorMsg != nil {
				defer C.free(unsafe.Pointer(errorMsg))
				return fmt.Errorf("[cgo onWrite failed: %v]", C.GoString(errorMsg))
			}
			return fmt.Errorf("[cgo onWrite failed with code: %v]", result)
		}

		return nil
	}

	module := qf.NewPlugin(moduleA.Name, moduleA.Desc, moduleA.Desc, serv, cfg, onWrite)
	onRead := module.Run().(qf.OnReadDelegate)

	onReadAdapter := func(data uintptr, length uintptr) uintptr {
		if data != 0 && length > 0 && onRead != nil {
			// 将 C 指针转换为 Go slice
			goBytes := C.GoBytes(unsafe.Pointer(data), C.int(length))
			// 调用真正的 onRead 函数
			onRead(goBytes)
		}
		return 0
	}

	// 将函数指针写入到传入的地址
	*(*uintptr)(unsafe.Pointer(onReadCallbackPtr)) = syscall.NewCallback(onReadAdapter)
}
