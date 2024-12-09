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

*/
import "C"
import (
	"encoding/json"
	"errors"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"unsafe"
)

func main() {

}

var (
	service              *qservice.MicroService
	onInitCallbackFunc   C.OnInitCallback
	onReqCallbackFunc    C.OnReqCallback
	onNoticeCallbackFunc C.OnNoticeCallback
)

//export Start
func Start(moduleName *C.char, moduleDesc *C.char, descLen C.int, version *C.char,
	customArgs *C.char, argsLen C.int, onInitCallback C.OnInitCallback, onReqCallback C.OnReqCallback, onNoticeCallback C.OnNoticeCallback) C.int {
	name := string(C.GoString(moduleName))
	desc := string(C.GoBytes(unsafe.Pointer(moduleDesc), descLen))
	ver := string(C.GoString(version))
	argStr := string(C.GoBytes(unsafe.Pointer(customArgs), argsLen))
	onInitCallbackFunc = onInitCallback
	onReqCallbackFunc = onReqCallback
	onNoticeCallbackFunc = onNoticeCallback

	// 创建微服务
	setting := qservice.NewSetting(name, desc, ver)
	if onInitCallback != nil {
		setting.BindInitFunc(onInit)
	}
	if onReqCallback != nil {
		setting.BindReqFunc(onReqHandler)
	}
	if onNoticeCallback != nil {
		setting.BindNoticeFunc(onNoticeHandler)
	}

	// 重新传入参数
	if argStr != "" {
		args := qservice.Args{}
		err := json.Unmarshal([]byte(argStr), &args)
		if err != nil {
			panic(err)
		}
		setting.ReloadByCustomArgs(args)
	}

	service = qservice.NewService(setting)

	// 启动微服务
	go service.Run()
	return C.int(1)
}

//export Stop
func Stop() {
	if service == nil {
		return
	}
	service.Stop()
}

//export SendRequest
func SendRequest(module, route *C.char, paramsJson *C.char, paramsLen C.int,
	respJson **C.char, respLength *C.int, errorStr **C.char, errorLen *C.int) C.int {

	var params any
	str := C.GoBytes(unsafe.Pointer(paramsJson), paramsLen)
	err := json.Unmarshal([]byte(string(str)), &params)
	if err != nil {
		data, _ := json.Marshal(err)
		*errorStr = (*C.char)(C.CBytes(data))
		*errorLen = C.int(len(data))
		return C.int(0)
	}

	moduleGo := C.GoString(module)
	routeGo := C.GoString(route)
	ctx, err := service.SendRequest(moduleGo, routeGo, params)
	if err != nil {
		data, _ := json.Marshal(err)
		*errorStr = (*C.char)(C.CBytes(data))
		*errorLen = C.int(len(data))
		return C.int(0)
	}

	data, _ := json.Marshal(ctx.Raw())
	*respJson = (*C.char)(C.CBytes(data))
	*respLength = C.int(len(data))

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
