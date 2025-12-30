package main

/*
#include <windows.h>

*/
import "C"
import (
	easyCon "github.com/qiu-tec/easy-con.golang"
	"syscall"
	"unsafe"
)

func main() {
	broker := easyCon.NewCgoBroker()

	dllHandle, err := syscall.LoadLibrary("../template/moduleA/build/plugin/moduleA.dll")
	if err != nil {
		panic(err)
	}
	defer syscall.FreeLibrary(dllHandle)

	initProc, err := syscall.GetProcAddress(dllHandle, "Init")
	if err != nil {
		panic(err)
	}

	callback := syscall.NewCallback(func(data uintptr, length uintptr, errorMsgPtr uintptr) uintptr {
		bytes := C.GoBytes(unsafe.Pointer(data), C.int(length))
		err = broker.Publish(bytes)
		if err != nil {
			// 将错误信息写回给C端
			errStr := C.CString(err.Error())
			// 将指针写入C端提供的地址
			*(*uintptr)(unsafe.Pointer(errorMsgPtr)) = uintptr(unsafe.Pointer(errStr))
			return 1
		}

		// 成功时清空错误信息
		*(*uintptr)(unsafe.Pointer(errorMsgPtr)) = 0
		return 0
	})

	// 用于接收返回的 onRead 函数指针
	var onReadFuncPtr uintptr

	// 调用Init方法，传入 onWrite 回调和接收 onRead 的指针地址
	_, _, _ = syscall.SyscallN(initProc, callback, uintptr(unsafe.Pointer(&onReadFuncPtr)))

	// 注册 onRead 到 broker
	broker.RegClient("ModuleA", func(data []byte) {
		// 将数据传递给 moduleA 返回的 onRead 函数指针
		cData := C.CBytes(data)
		defer C.free(unsafe.Pointer(cData))

		// 调用 moduleA 返回的 onRead 函数指针
		syscall.Syscall(onReadFuncPtr, uintptr(cData), uintptr(len(data)), 0, 0)
	})

	select {}
}
