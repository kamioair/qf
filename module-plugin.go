package qf

/*
#include <string.h>
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <windows.h>

// OnWrite函数类型定义
typedef int (*OnWriteCallback)(char*, int, char**);

// OnWriteHandler包装函数
static int OnWriteHandler(OnWriteCallback cb, char* respBytes, int respLen, char** outErrorMsg) {
	int result = cb(respBytes, respLen, outErrorMsg);
	return result;
}

*/
import "C"

import (
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"syscall"
	"time"
	"unsafe"
)

type plugin struct {
	*baseModule       // 嵌入基础模块
	onWrite           OnWriteDelegate
	onRead            OnReadDelegate
	onReadCallbackPtr uintptr
}

// NewPlugin 创建CGo插件模块
// onWriteCallback: C端的写入回调函数指针
// onReadCallbackPtr: 用于返回Go端的读取回调函数指针的地址
func NewPlugin(
	service IService,
	onWriteCallback uintptr,
	onReadCallbackPtr uintptr,
) IModule {
	// 创建onWrite适配器
	onWrite := createOnWriteAdapter(onWriteCallback)

	p := &plugin{
		baseModule:        newBaseModule(service),
		onWrite:           onWrite,
		onReadCallbackPtr: onReadCallbackPtr,
	}

	// 创建onRead适配器并设置函数指针
	setOnReadAdapter(p.onRead, p.onReadCallbackPtr)

	return p
}

// 内部测试用
func newPluginTest(service IService, onWrite OnWriteDelegate) *plugin {
	p := &plugin{
		baseModule: newBaseModule(service),
		onWrite:    onWrite,
	}
	return p
}

// Name 获取模块名称
func (p *plugin) Name() string {
	return p.service.config().getBase().module
}

// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
func (p *plugin) Run() {
	cfg := p.service.config().getBase()

	defer errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	}, cfg.module, "init", nil)

	// 打印模块信息
	p.printModuleInfo()

	name := cfg.module
	if cfg.Broker.IsRandomClientID {
		name = fmt.Sprintf("%s-%d", cfg.module, time.Now().UnixNano())
	}

	setting := easyCon.CoreSetting{
		Module:            name,
		TimeOut:           time.Duration(cfg.Broker.TimeOut) * time.Millisecond,
		ReTry:             cfg.Broker.Retry,
		LogMode:           easyCon.ELogMode(cfg.Broker.LogMode),
		PreFix:            cfg.Broker.Prefix,
		ChannelBufferSize: cfg.Broker.ChannelBufferSize,
		ConnectRetryDelay: time.Duration(cfg.Broker.ConnectRetryDelay) * time.Millisecond,
		IsWaitLink:        cfg.Broker.LinkTimeOut == 0,
		IsSync:            false,
	}

	// 构建回调
	callback := p.buildAdapterCallBack(p.onState, p.onReq, p.onExiting, p.getVersion)

	// 启动客户端
	p.adapter, p.onRead = easyCon.NewCgoAdapter(setting, callback, p.onWrite)

	// 调用业务的初始化
	p.callOnInit()

	// 保存配置文件
	p.saveConfig()

	// 启动成功
	fmt.Printf("\nStart OK\n\n")
}

// RunAsync 异步运行模块，执行后不等待
func (p *plugin) RunAsync() {
	p.Run()
}

func (p *plugin) Stop() {
	// 调用业务的退出
	p.callOnStop()
	// 退出客户端
	p.stopAdapter()
}

func (p *plugin) onExiting() {
	// Plugin模式下不需要退出
}

func (p *plugin) onState(status easyCon.EStatus) {
	p.callOnState(status)
}

func (p *plugin) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp []byte) {
	return p.handleReq(pack, p.Stop)
}

// createOnWriteAdapter 创建写入适配器
func createOnWriteAdapter(onWriteCallbackPtr uintptr) OnWriteDelegate {
	return func(data []byte) error {
		onWriteCallback := (C.OnWriteCallback)(unsafe.Pointer(onWriteCallbackPtr))

		size := C.int(len(data))
		packStr := (*C.char)(C.CBytes(data))

		// 创建错误信息指针变量，用于接收回调返回的错误
		var errorMsg *C.char

		// 调用回调函数，将 errorMsg 的地址传递给 C 函数
		result := C.OnWriteHandler(onWriteCallback, packStr, size, &errorMsg)

		// 释放通过 C.CBytes 分配的内存
		C.free(unsafe.Pointer(packStr))

		// 检查返回结果
		if result != 0 {
			if errorMsg != nil {
				defer C.free(unsafe.Pointer(errorMsg))
				return fmt.Errorf("[cgo onWrite failed: %v]", C.GoString(errorMsg))
			}
			return fmt.Errorf("[cgo onWrite failed with code: %v]", result)
		}

		return nil
	}
}

// setOnReadAdapter 创建并设置读取适配器
func setOnReadAdapter(onRead OnReadDelegate, onReadCallbackPtr uintptr) {
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
