package qservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconvert"
	"github.com/kamioair/qf/utils/qio"
	"github.com/kamioair/qf/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"os"
	"strings"
	"time"
)

type MicroService struct {
	adapter       easyCon.IAdapter
	setting       *Setting
	brokerModules map[string]string
}

// NewService 创建服务
func NewService(setting *Setting) *MicroService {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	// 创建服务
	serv := &MicroService{
		setting:       setting,
		brokerModules: map[string]string{},
	}

	// 连接
	serv.initAdapter()

	return serv
}

// Run 启动服务
func (serv *MicroService) Run() {
	qlauncher.Run(serv.onStart, serv.onStop)
}

// Stop 停止
func (serv *MicroService) Stop() {
	serv.onStop()
	qlauncher.Exit()
}

// Setting 获取设置
func (serv *MicroService) Setting() Setting {
	return Setting{
		Mode:    serv.setting.Mode,
		Module:  serv.setting.Module,
		Desc:    serv.setting.Desc,
		Version: serv.setting.Version,
		DevCode: serv.setting.DevCode,
		Broker:  serv.setting.Broker,
	}
}

// Adapter 获取访问器
func (serv *MicroService) Adapter() easyCon.IAdapter {
	return serv.adapter
}

// ResetClient 重置客户端
func (serv *MicroService) ResetClient() {
	// 重新创建服务
	serv.initAdapter()
}

// SendRequest 向服务器其他模块发送请求，单机两者效果一致
func (serv *MicroService) SendRequest(module, route string, params any) (qdefine.Context, error) {
	var resp easyCon.PackResp

	if strings.Contains(module, "/") {
		// 路由请求
		newParams := map[string]any{}
		newParams["Module"] = module
		newParams["Route"] = route
		newParams["Content"] = params
		resp = serv.adapter.Req(routeModuleName, "Request", newParams)
	} else {
		// 常规请求，查找请求模块所在设备ID
		devCode := serv.setting.DevCode
		if code, ok := serv.brokerModules[module]; ok {
			devCode = code
		}
		if module == routeModuleName && serv.setting.Mode.IsServer() {
			devCode = ""
		}
		resp = serv.adapter.Req(serv.newModuleName(module, devCode), route, params)
	}
	if resp.RespCode == easyCon.ERespSuccess {
		// 返回成功
		return newContentByResp(resp)
	}
	// 返回异常
	if resp.RespCode == easyCon.ERespTimeout {
		return nil, errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request timeout"))
	}
	if resp.RespCode == easyCon.ERespRouteNotFind {
		return nil, errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request route not find"))
	}
	if resp.RespCode == easyCon.ERespForbidden {
		return nil, errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request forbidden"))
	}
	return nil, errors.New(fmt.Sprintf("%v:%s,%s", resp.RespCode, resp.Content, resp.Error))
}

//// SendStatus 发送状态消息
//func (serv *MicroService) SendStatus(route string, content map[string]any) error {
//	arg := map[string]map[string]any{}
//	arg[route] = content
//	resp := serv.adapter.Req(routeModuleName, "StatusInput", arg)
//	if resp.RespCode == easyCon.ERespSuccess {
//		// 返回成功
//		return nil
//	}
//	// 返回异常
//	if resp.RespCode == easyCon.ERespTimeout {
//		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request timeout"))
//	}
//	if resp.RespCode == easyCon.ERespRouteNotFind {
//		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request route not find"))
//	}
//	if resp.RespCode == easyCon.ERespForbidden {
//		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request forbidden"))
//	}
//	return errors.New(fmt.Sprintf("%v:%s,%s", resp.RespCode, resp.Content, resp.Error))
//}

func (serv *MicroService) SendAlarm(alarmType string, alarmValue string) error {
	// 客户端模式往客户端路由模块发送，反之往根路由发送
	devCode := serv.setting.DevCode
	if serv.setting.Mode.IsServer() {
		devCode = ""
	}
	params := map[string]string{
		"Type":  alarmType,
		"Value": alarmValue,
	}
	resp := serv.adapter.Req(serv.newModuleName(routeModuleName, devCode), "CustomAlarm", params)
	if resp.RespCode == easyCon.ERespSuccess {
		// 返回成功
		return nil
	}
	// 返回异常
	if resp.RespCode == easyCon.ERespTimeout {
		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request timeout"))
	}
	if resp.RespCode == easyCon.ERespRouteNotFind {
		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request route not find"))
	}
	if resp.RespCode == easyCon.ERespForbidden {
		return errors.New(fmt.Sprintf("%v:%s", resp.RespCode, "request forbidden"))
	}
	return errors.New(fmt.Sprintf("%v:%s,%s", resp.RespCode, resp.Content, resp.Error))
}

// SendNotice 发送通知
func (serv *MicroService) SendNotice(route string, content any) {
	err := serv.adapter.SendNotice(route, content)
	if err != nil {
		serv.SendLog("error", "Service Send Notice Error", err)
	}
}

// SendLog 发送日志
func (serv *MicroService) SendLog(logType qdefine.ELog, content string, err error) {
	log := runLog{
		Id:      serv.setting.DevCode,
		Module:  serv.setting.Module,
		Content: content,
	}
	js, _ := json.Marshal(log)
	switch logType {
	case qdefine.ELogError:
		serv.adapter.Err(string(js), err)
		writeErrLog(serv.setting.Module, content, err.Error())
	case qdefine.ELogWarn:
		serv.adapter.Warn(string(js))
	case qdefine.ELogDebug:
		serv.adapter.Debug(string(js))
	}
}

func (serv *MicroService) initAdapter() {
	// 如果之前连接了，则先停止
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
	// 创建连接
	newName := serv.newModuleName(serv.setting.Module, serv.setting.DevCode)
	if serv.setting.Module == routeModuleName && serv.setting.Mode.IsServer() {
		newName = serv.setting.Module
	}
	apiSetting := easyCon.NewSetting(newName, serv.setting.Broker.Addr, serv.onReq, serv.onStatusChanged)
	if serv.setting.onNoticeHandler != nil {
		apiSetting.OnNotice = serv.onNotice
	}
	if serv.setting.onStatusHandler != nil {
		apiSetting.OnRetainNotice = serv.onRetainNotice
	}
	if serv.setting.onAcceptDetectedHandler != nil {
		apiSetting.OnRespDetected = serv.onRespDetected
	}
	apiSetting.UID = serv.setting.Broker.UId
	apiSetting.PWD = serv.setting.Broker.Pwd
	apiSetting.TimeOut = time.Duration(serv.setting.Broker.TimeOut) * time.Millisecond
	apiSetting.ReTry = serv.setting.Broker.Retry
	apiSetting.LogMode = easyCon.ELogMode(serv.setting.Broker.LogMode)
	apiSetting.DetectedRoutes = serv.setting.DetectedRoutes
	apiSetting.WatchedModules = serv.setting.WatchedModules
	serv.adapter = easyCon.NewMqttAdapter(apiSetting)

	// 等待确保连接成功
	time.Sleep(time.Millisecond * 50)
}

func (serv *MicroService) knockDoor() {
	if serv.setting.DevCode == "" || strings.HasSuffix(serv.setting.DevCode, "[TEMP]") {
		// 单机模式不敲门
		return
	}
	// 向同级路由模块发送敲门
	info := map[string]any{}
	info["Id"] = serv.setting.DevCode
	info["Modules"] = []map[string]string{
		{
			"Name":    serv.setting.Module,
			"Desc":    serv.setting.Desc,
			"Version": serv.setting.Version,
		},
	}
	name := routeModuleName
	if serv.setting.Mode == EModeClient {
		name = serv.newModuleName(routeModuleName, serv.setting.DevCode)
	}
	full := map[string]map[string]any{}
	full[serv.setting.DevCode] = info
	ctx, err := serv.SendRequest(name, "KnockDoor", full)
	if err != nil {
		panic(err)
	}
	// 获取路由模块返回的服务端模块列表
	serv.brokerModules = qconvert.ToAny[map[string]string](ctx.Raw())
}

func (serv *MicroService) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = err
		// 记录日志
		writeErrLog(serv.setting.Module, "service.onReq", err)
	})

	switch pack.Route {
	case "Exit":
		serv.onStop()
		go func() {
			time.Sleep(time.Millisecond * 100)
			qlauncher.Exit()
		}()
		return easyCon.ERespSuccess, nil
	case "Reset":
		serv.adapter.Reset()
		return easyCon.ERespSuccess, nil
	}
	if serv.setting.onReqHandler != nil {
		ctx, err1 := newContentByReq(pack)
		if err1 != nil {
			return easyCon.ERespError, err1.Error()
		}
		rs, err2 := serv.setting.onReqHandler(pack.Route, ctx)
		if err2 != nil {
			return easyCon.ERespError, err2.Error()
		}
		// 执行成功，返回结果
		return easyCon.ERespSuccess, rs
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (serv *MicroService) onNotice(notice easyCon.PackNotice) {
	defer errRecover(func(err string) {
		// 记录日志
		writeErrLog(serv.setting.Module, "service.onNotice", err)
	})

	// 外置方法
	if serv.setting.onNoticeHandler != nil {
		ctx, err := newContentByNotice(notice)
		if err != nil {
			panic(err)
		}
		serv.setting.onNoticeHandler(notice.Route, ctx)
	}
}

func (serv *MicroService) onRetainNotice(notice easyCon.PackNotice) {
	defer errRecover(func(err string) {
		// 记录日志
		writeErrLog(serv.setting.Module, "service.onNotice", err)
	})

	//if notice.Route == "GlobalStatusRetain" {
	//	content := map[string]any{}
	//	str, _ := json.Marshal(notice.Content)
	//	_ = json.Unmarshal(str, &content)
	//
	//	// 外置方法
	//	if serv.setting.onStatusHandler != nil {
	//		for k, v := range content {
	//			ctx, err := newContentByData(v)
	//			if err != nil {
	//				panic(err)
	//			}
	//			serv.setting.onStatusHandler(k, ctx)
	//		}
	//	}
	//}
}

func (serv *MicroService) onStatusChanged(adapter easyCon.IAdapter, status easyCon.EStatus) {
	if serv.setting.onCommStateHandler != nil {
		sn := qdefine.ECommState(status)
		serv.setting.onCommStateHandler(sn)
	}
}

func (serv *MicroService) onRespDetected(pack easyCon.PackResp) {
	if serv.setting.onAcceptDetectedHandler == nil {
		return
	}
	ctx, err := newContentByResp(pack)
	if err != nil {
		panic(err)
	}
	serv.setting.onAcceptDetectedHandler(pack.From, pack.Route, ctx)
}

func (serv *MicroService) onStart() {
	// 执行外部初始化
	if serv.setting.onInitHandler != nil {
		serv.setting.onInitHandler(serv.setting.Module)
	}

	serv.knockDoor()
}

func (serv *MicroService) onStop() {
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
}

func (serv *MicroService) newModuleName(module, code string) string {
	sp := strings.Split(module, ".")
	if len(sp) >= 2 {
		module = sp[0] + "." + code
	} else {
		module = module + "." + code
	}
	return strings.Trim(module, ".")
}
