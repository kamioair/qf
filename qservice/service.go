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
	"strconv"
	"strings"
	"time"
)

type MicroService struct {
	adapter       easyCon.IAdapter
	setting       *Setting
	retainContent map[string]any
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
		retainContent: make(map[string]any),
	}

	// 启动访问器
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
func (serv *MicroService) ResetClient(code string) {
	serv.setting.SetDeviceCode(code)
	// 重新创建服务
	serv.initAdapter()
}

func (serv *MicroService) newModuleName(module, code string) string {
	if strings.Contains(module, ".") {
		return module
	}
	str := module + "." + code
	return strings.Trim(str, ".")
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
		// 常规请求
		resp = serv.adapter.Req(module, route, params)
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

// SendNoticeRetain 发送Retain消息
func (serv *MicroService) SendNoticeRetain(route string, content any) error {
	serv.retainContent[route] = content
	return serv.adapter.SendRetainNotice("GlobalRetainNotice", serv.retainContent)
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
	// 先停止
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
	// 重新创建
	newName := newModuleName(serv.setting.Module, serv.setting.DevCode)
	apiSetting := easyCon.NewSetting(newName, serv.setting.Broker.Addr, serv.onReq, serv.onStatusChanged)
	apiSetting.OnNotice = serv.onNotice
	apiSetting.OnRetainNotice = serv.onRetainNotice
	apiSetting.UID = serv.setting.Broker.UId
	apiSetting.PWD = serv.setting.Broker.Pwd
	apiSetting.TimeOut = time.Duration(serv.setting.Broker.TimeOut) * time.Second
	apiSetting.ReTry = serv.setting.Broker.Retry
	apiSetting.LogMode = easyCon.ELogMode(serv.setting.Broker.LogMode)
	serv.adapter = easyCon.NewMqttAdapter(apiSetting)

	// 如果是路由模式，则向上级自报家门
	if _, ok := strconv.Atoi(serv.setting.DevCode); ok == nil {
		info := map[string]any{}
		info["Id"] = serv.setting.DevCode
		info["Name"] = serv.setting.DevName
		info["Modules"] = []map[string]string{
			{
				"Name":    serv.setting.Module,
				"Desc":    serv.setting.Desc,
				"Version": serv.setting.Version,
			},
		}
		_, _ = serv.SendRequest(clientModuleName, "KnockDoor", info)
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

	if notice.Route == "GlobalRetainNotice" {
		str, _ := json.Marshal(notice.Content)
		_ = json.Unmarshal(str, &serv.retainContent)

		// 外置方法
		if serv.setting.onRetainNoticeHandler != nil {
			for k, v := range serv.retainContent {
				ctx, err := newContentByData(v)
				if err != nil {
					panic(err)
				}
				serv.setting.onRetainNoticeHandler(k, ctx)
			}
		}
	}
}

func (serv *MicroService) onStatusChanged(adapter easyCon.IAdapter, status easyCon.EStatus) {
	//if status == easyCon.EStatusLinkLost {
	//	adapter.Reset()
	//}
	if serv.setting.onStateHandler != nil {
		sn := qdefine.ECommState(status)
		serv.setting.onStateHandler(sn)
	}
}

func (serv *MicroService) onStart() {
	if serv.setting.onInitHandler != nil {
		serv.setting.onInitHandler(serv.setting.Module)
	}
}

func (serv *MicroService) onStop() {
	if serv.adapter != nil {
		serv.adapter.Stop()
		serv.adapter = nil
	}
}
