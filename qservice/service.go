package qservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qio"
	"github.com/kamioair/qf/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MicroService struct {
	adapter       easyCon.IAdapter
	setting       *Setting
	reqDeviceDict map[string]string
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
		reqDeviceDict: map[string]string{},
	}

	// 连接
	serv.initAdapter()

	return serv
}

// Run 启动服务
func (serv *MicroService) Run() {
	qlauncher.Run(serv.onStart, serv.onStop)
}

func (serv *MicroService) Setting() Setting {
	return Setting{
		Module:  serv.setting.Module,
		Desc:    serv.setting.Desc,
		Version: serv.setting.Version,
		DevCode: serv.setting.DevCode,
		Broker:  serv.setting.Broker,
	}
}

// ResetClient 重置客户端
func (serv *MicroService) ResetClient(devCode string) {
	serv.setting.SetDeviceCode(devCode)
	// 重新创建服务
	serv.initAdapter()
	// 向Broker敲门
	serv.KnockDoor()
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
		// 常规请求，先从服务请求设备列表，发送模块请求是附带设备ID
		// 如果没有则全部请求到本电脑的模块
		devCode := serv.setting.DevCode
		if code, ok := serv.reqDeviceDict[module]; ok {
			devCode = code
		}
		resp = serv.adapter.Req(newModuleName(module, devCode), route, params)
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

// SendStatus 发送状态消息
func (serv *MicroService) SendStatus(route string, content map[string]any) error {
	arg := map[string]map[string]any{}
	arg[route] = content
	resp := serv.adapter.Req(routeModuleName, "StatusInput", arg)
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
	switch logType {
	case qdefine.ELogError:
		serv.adapter.Err(content, err)
	case qdefine.ELogWarn:
		serv.adapter.Warn(content)
	case qdefine.ELogDebug:
		serv.adapter.Debug(content)
	default:
		serv.adapter.Debug(content)
	}
}

func (serv *MicroService) initAdapter() {
	// 如果之前连接了，则先停止
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
	// 创建连接
	devCode := ""
	if serv.setting.RouteMode == "client" {
		devCode = serv.setting.DevCode
	}
	newName := newModuleName(serv.setting.Module, devCode)
	apiSetting := easyCon.NewSetting(newName, serv.setting.Broker.Addr, serv.onReq, serv.onStatusChanged)
	apiSetting.OnNotice = serv.onNotice
	apiSetting.OnRetainNotice = serv.onRetainNotice
	apiSetting.UID = serv.setting.Broker.UId
	apiSetting.PWD = serv.setting.Broker.Pwd
	apiSetting.TimeOut = time.Duration(serv.setting.Broker.TimeOut) * time.Second
	apiSetting.ReTry = serv.setting.Broker.Retry
	apiSetting.LogMode = easyCon.ELogMode(serv.setting.Broker.LogMode)
	serv.adapter = easyCon.NewMqttAdapter(apiSetting)

	// 等待确保连接成功
	time.Sleep(time.Second)

	// 问主路由模块请求服务器的模块列表和本机的所有模块列表
	serv.loadServDevices()
}

func (serv *MicroService) loadServDevices() {
	if serv.setting.DevCode == "" {
		return
	}
	resp := serv.adapter.Req(routeModuleName, "LoadServDevices", serv.setting.DevCode)
	if resp.RespCode == easyCon.ERespSuccess {
		err := json.Unmarshal(resp.Content.([]byte), &serv.reqDeviceDict)
		if err != nil {
			writeErrLog("service.loadServDevices json error", err.Error())
		}
	} else {
		writeErrLog("service.loadServDevices req error", fmt.Sprintf("%s,%s", resp.RespCode, resp.Error))
	}
}

func (serv *MicroService) KnockDoor() {
	devCode := serv.setting.DevCode
	parent := ""
	if serv.setting.RouteMode == "client" {
		sp := strings.Split(devCode, "_")
		if len(sp) > 1 {
			parent = strings.Join(sp[:len(sp)-1], "_")
		}
	}
	// 非单机模式，向Broker所在路由敲门
	if devCode != "" && isUUID(devCode) == false {
		info := map[string]any{}
		info["Id"] = devCode
		info["Name"] = serv.setting.DevName
		info["Parent"] = parent
		info["Modules"] = []map[string]string{
			{
				"Name":    serv.setting.Module,
				"Desc":    serv.setting.Desc,
				"Version": serv.setting.Version,
			},
		}
		_, _ = serv.SendRequest(routeModuleName, "KnockDoor", info)
	}
}

func (serv *MicroService) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = err
		// 记录日志
		writeErrLog("service.onReq", err)
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
			c, _ := strconv.Atoi(err2.Error())
			switch c {
			case int(easyCon.ERespBadReq):
				return easyCon.ERespBadReq, "request bad"
			case int(easyCon.ERespRouteNotFind):
				return easyCon.ERespRouteNotFind, "request route not find"
			case int(easyCon.ERespForbidden):
				return easyCon.ERespForbidden, "request forbidden"
			case int(easyCon.ERespTimeout):
				return easyCon.ERespTimeout, "request timeout"
			default:
				return easyCon.ERespError, err2.Error()
			}
		}
		// 执行成功，返回结果
		return easyCon.ERespSuccess, rs
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (serv *MicroService) onNotice(notice easyCon.PackNotice) {
	defer errRecover(func(err string) {
		// 记录日志
		writeErrLog("service.onNotice", err)
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
		writeErrLog("service.onNotice", err)
	})

	if notice.Route == "GlobalStatusRetain" {
		content := map[string]any{}
		str, _ := json.Marshal(notice.Content)
		_ = json.Unmarshal(str, &content)

		// 外置方法
		if serv.setting.onStatusHandler != nil {
			for k, v := range content {
				ctx, err := newContentByData(v)
				if err != nil {
					panic(err)
				}
				serv.setting.onStatusHandler(k, ctx)
			}
		}
	}
}

func (serv *MicroService) onStatusChanged(adapter easyCon.IAdapter, status easyCon.EStatus) {
	//if status == easyCon.EStatusLinkLost {
	//	adapter.Reset()
	//}
	if serv.setting.onCommStateHandler != nil {
		sn := qdefine.ECommState(status)
		serv.setting.onCommStateHandler(sn)
	}
}

func (serv *MicroService) onStart() {
	// 执行外部初始化
	if serv.setting.onInitHandler != nil {
		serv.setting.onInitHandler(serv.setting.Module)
	}

	// 敲门
	serv.KnockDoor()
}

func (serv *MicroService) onStop() {
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
}

func (serv *MicroService) newModuleName(module, code string) string {
	if strings.Contains(module, ".") {
		return module
	}
	str := module + "." + code
	return strings.Trim(str, ".")
}

func isUUID(uuid string) bool {
	r, _ := regexp.Compile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}
