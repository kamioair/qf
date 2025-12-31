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
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"syscall"
	"time"
	"unsafe"
)

type plugin struct {
	service           IService
	reg               *Reg
	adapter           easyCon.IAdapter
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
		service:           service,
		onWrite:           onWrite,
		onReadCallbackPtr: onReadCallbackPtr,
	}

	// 注册方法
	p.reg = &Reg{}
	p.service.Reg(p.reg)

	return p
}

func (p *plugin) Run() {
	cfg := p.service.config().getBase()

	defer errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	}, cfg.module, "init", nil)

	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")

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
		ChannelBufferSize: 100,
		ConnectRetryDelay: time.Second,
		IsWaitLink:        cfg.Broker.LinkTimeOut == 0,
		IsSync:            false,
	}

	callback := easyCon.AdapterCallBack{
		OnStatusChanged: p.onState,
		OnReqRec:        p.onReq,
		OnRespRec:       nil,
		OnExiting:       p.onExiting,
		OnGetVersion:    p.onGetVersion,
	}
	if p.reg.OnNotice != nil {
		callback.OnNoticeRec = p.reg.OnNotice
	}
	if p.reg.OnRetainNotice != nil {
		callback.OnRetainNoticeRec = p.reg.OnRetainNotice
	}
	if p.reg.OnLog != nil {
		callback.OnLogRec = p.reg.OnLog
	}

	p.adapter, p.onRead = easyCon.NewCgoAdapter(setting, callback, p.onWrite)

	// 调用业务的初始化
	p.service.setEnv(p.reg, p.adapter, nil)
	if p.reg.OnInit != nil {
		p.reg.OnInit()
	}

	// 保存配置文件
	saveConfigFile(cfg)

	// 创建onRead适配器并设置函数指针
	setOnReadAdapter(p.onRead, p.onReadCallbackPtr)

	// 启动成功
	fmt.Printf("\nStart OK\n\n")
}

func (p *plugin) Stop() {

}

func (p *plugin) onExiting() {
	// Plugin模式下不需要退出
}

func (p *plugin) onState(status easyCon.EStatus) {
	fmt.Printf("Client link state = [%s]\n", status)

	if p.reg != nil && p.reg.OnStatusChanged != nil {
		go p.reg.OnStatusChanged(status)
	}
}

func (p *plugin) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	cfg := p.service.config().getBase()

	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = errors.New(err)
	}, cfg.module, pack.Route, pack.Content)

	switch pack.Route {
	case "Exit":
		p.Stop()
		return easyCon.ERespSuccess, nil
	case "Version":
		ver := map[string]string{}
		ver["Module"] = cfg.module
		ver["Desc"] = cfg.desc
		ver["ModuleVersion"] = cfg.version
		ver["FrameVersion"] = Version
		return easyCon.ERespSuccess, ver
	}
	if p.reg.OnReq != nil {
		code, resp = p.reg.OnReq(pack)
		if code != easyCon.ERespSuccess {
			// 记录日志
			str, _ := json.Marshal(pack.Content)
			errStr := ""
			if e := resp.(error); e != nil {
				errStr = e.Error()
			} else {
				errStr = fmt.Sprintf("%v", resp)
			}
			writeLog(cfg.module, "Error", fmt.Sprintf("[OnReq From %s.%s] InParam=%s", pack.From, pack.Route, str), formatRespError(code, errStr))
		}
		return code, resp
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (p *plugin) onGetVersion() []string {
	cfg := p.service.config().getBase()
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", cfg.version)}
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
