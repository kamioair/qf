package main

/*
#include <string.h>
#include <stdio.h>
#include <stdint.h>

typedef void (*OnInitCallback)(char*);
static void InitHandler(OnInitCallback cb, char* moduleName) {
	cb(moduleName);
}

typedef int (*OnReqCallback)(char*, char*, int, char**, int*, char**);
static int ReqHandler(OnReqCallback cb, char* route, char* paramStr, int paramLen, char** respJson, int* respLen, char** error) {
	return cb(route, paramStr, paramLen, respJson, respLen, error);
}

typedef int (*OnNoticeCallback)(char*, char*, int);
static int NoticeHandler(OnNoticeCallback cb, char* route, char* paramStr, int paramLen) {
	return cb(route, paramStr, paramLen);
}

typedef int (*OnCommStateCallback)(char*);
static int CommStateeHandler(OnCommStateCallback cb, char* state) {
	return cb(state);
}

*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qio"
	"unsafe"
)

func main() {

}

var (
	service                 *qservice.MicroService
	onInitCallbackFunc      C.OnInitCallback
	onReqCallbackFunc       C.OnReqCallback
	onNoticeCallbackFunc    C.OnNoticeCallback
	onCommStateCallbackFunc C.OnCommStateCallback
)

type startArg struct {
	Module     string
	Desc       string
	Version    string
	CustomArgs qservice.Args
}

//export Start
func Start(settingJson *C.char, settingLen C.int, onInitCallback C.OnInitCallback, onCommStateCallback C.OnCommStateCallback,
	onReqCallback C.OnReqCallback, onNoticeCallback C.OnNoticeCallback) {

	args := startArg{}
	str := C.GoBytes(unsafe.Pointer(settingJson), settingLen)
	err := json.Unmarshal([]byte(string(str)), &args)
	if err != nil {
		panic(err)
	}

	qio.WriteString(".\\log.txt", string(str), true)

	onInitCallbackFunc = onInitCallback
	onReqCallbackFunc = onReqCallback
	onNoticeCallbackFunc = onNoticeCallback
	onCommStateCallbackFunc = onCommStateCallback

	qio.WriteString(".\\log.txt", fmt.Sprintln(args.Module, args.Desc, args.Version), true)

	// 创建微服务
	setting := qservice.NewSetting(args.Module, args.Desc, args.Version)
	if onInitCallback != nil {
		setting.BindInitFunc(onInit)
	}
	if onReqCallback != nil {
		setting.BindReqFunc(onReqHandler)
	}
	if onNoticeCallback != nil {
		setting.BindNoticeFunc(onNoticeHandler)
	}
	if onCommStateCallback != nil {
		setting.BindCommStateFunc(onCommStateHandler)
	}

	// 重新设置参数
	setting.ReloadByCustomArgs(args.CustomArgs)

	// 启动微服务
	service = qservice.NewService(setting)
	service.Run()
}

//export Stop
func Stop() {
	if service == nil {
		return
	}
	service.Stop()
}

type Resp struct {
	Content any
	Error   string
}

//export SendRequest
func SendRequest(module, route *C.char, paramsJson *C.char, paramsLen C.int, respJson **C.char, respLength *C.int) C.int {

	resp := Resp{}

	var params any
	str := C.GoBytes(unsafe.Pointer(paramsJson), paramsLen)
	err := json.Unmarshal([]byte(string(str)), &params)
	if err != nil {
		resp.Error = err.Error()
	}

	// 执行请求
	moduleGo := string(C.GoString(module))
	routeGo := string(C.GoString(route))
	ctx, err := service.SendRequest(moduleGo, routeGo, params)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Content = ctx.Raw()
	}

	// 转为Json
	data, _ := json.Marshal(resp)
	*respJson = (*C.char)(C.CBytes(data))
	*respLength = C.int(len(data))

	if resp.Error != "" {
		return C.int(0)
	}
	return C.int(1)
}

//export SendNotice
func SendNotice(route *C.char, paramsJson *C.char, paramsLen C.int) C.int {

	var params any
	str := C.GoBytes(unsafe.Pointer(paramsJson), paramsLen)
	err := json.Unmarshal([]byte(string(str)), &params)
	if err != nil {
		return C.int(0)
	}

	service.SendNotice(C.GoString(route), params)

	return C.int(1)
}

//export SendLog
func SendLog(logType *C.char, content *C.char, contentLength C.int, error *C.char, errorLength C.int) C.int {
	contentStr := string(C.GoBytes(unsafe.Pointer(content), contentLength))
	errorStr := string(C.GoBytes(unsafe.Pointer(error), errorLength))

	service.SendLog(qdefine.ELog(C.GoString(logType)), contentStr, errors.New(errorStr))
	return C.int(1)
}

//export SendAlarm
func SendAlarm(alarmType *C.char, typeLen C.int, alarmValue *C.char, valueLen C.int) C.int {
	typeStr := string(C.GoBytes(unsafe.Pointer(alarmType), typeLen))
	valueStr := string(C.GoBytes(unsafe.Pointer(alarmValue), valueLen))
	err := service.SendAlarm(typeStr, valueStr)
	if err != nil {
		return C.int(0)
	}
	return C.int(1)
}

func onNoticeHandler(route string, ctx qdefine.Context) {
	if onNoticeCallbackFunc == nil {
		return
	}

	data, _ := json.Marshal(ctx.Raw())
	size := C.int(len(data))
	packStr := (*C.char)(C.CBytes(data))

	C.NoticeHandler(onNoticeCallbackFunc, C.CString(route), packStr, size)
}

func onCommStateHandler(state qdefine.ECommState) {
	if onCommStateCallbackFunc != nil {
		C.CommStateeHandler(onCommStateCallbackFunc, C.CString(string(state)))
	}
}

func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	if onReqCallbackFunc == nil {
		return nil, errors.New("onReqCallbackFunc is nil")
	}

	data, _ := json.Marshal(ctx.Raw())
	size := C.int(len(data))
	packStr := (*C.char)(C.CBytes(data))

	var respLength C.int
	var respJson *C.char
	var respError *C.char
	rs := C.ReqHandler(onReqCallbackFunc, C.CString(route), packStr, size, &respJson, &respLength, &respError)
	if rs == 0 {
		return nil, errors.New(C.GoString(respError))
	}

	// Json反转...
	var obj any
	str := C.GoBytes(unsafe.Pointer(respJson), respLength)
	err := json.Unmarshal([]byte(string(str)), &obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func onInit(moduleName string) {
	if onInitCallbackFunc != nil {
		C.InitHandler(onInitCallbackFunc, C.CString(moduleName))
	}
}
