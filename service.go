package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qtime"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type Service struct {
	moduleName    string
	moduleDesc    string
	moduleVersion string
	adapter       easyCon.IAdapter
	cfg           IConfig
	reg           *Reg
	callback      CallbackDelegate
}

// GetRegEvents 获取注册绑定事件
func (bll *Service) GetRegEvents() *Reg {
	return bll.reg
}

// Load 初始化
func (bll *Service) Load(name, desc, version string, config IConfig, customSetting map[string]any) {
	bll.cfg = config
	if bll.cfg == nil {
		bll.cfg = &emptyConfig{}
	}
	// 设置模块信息
	bll.cfg.setBase(name, desc, version)
	// 加载配置
	loadConfig(bll.cfg, customSetting)
}

// NoticeInvoke 调用通知实现方法
func (bll *Service) NoticeInvoke(pack easyCon.PackNotice, onReq OnNoticeFunc) {
	ctx, err := newContent(pack.Content, nil, nil, &pack)
	if err != nil {
		bll.SendLogError(fmt.Sprintln("NoticeInvoke build invoke error", pack), err)
	}
	onReq(ctx)
}

// ReturnOk 返回成功
func (bll *Service) ReturnOk(content any) (code easyCon.EResp, resp any) {
	return easyCon.ERespSuccess, content
}

// ReturnErr 返回错误
func (bll *Service) ReturnErr(content any) (code easyCon.EResp, resp any) {
	js, _ := json.Marshal(content)
	return easyCon.ERespError, errors.New(string(js))
}

// ReturnNotFind 返回未找到
func (bll *Service) ReturnNotFind() (code easyCon.EResp, resp any) {
	return easyCon.ERespRouteNotFind, nil
}

// SendRequest 发送请求
func (bll *Service) SendRequest(module, route string, params any) (IContext, error) {
	resp := bll.adapter.Req(module, route, params)
	if resp.RespCode == easyCon.ERespSuccess {
		return newContent(resp.Content, nil, &resp, nil)
	}
	// 记录日志
	str, _ := json.Marshal(params)
	err := errors.New(formatRespError(resp.RespCode, resp.Error))
	bll.SendLogError(fmt.Sprintf("[SendRequest To %s.%s] InParams=%s", module, route, string(str)), err)
	return nil, err
}

// SendRequestWithTimeout 发送请求(可自定义超时时间的,单位毫秒)
func (bll *Service) SendRequestWithTimeout(module, route string, params any, timeout int) (IContext, error) {
	resp := bll.adapter.ReqWithTimeout(module, route, params, timeout)
	if resp.RespCode == easyCon.ERespSuccess {
		return newContent(resp.Content, nil, &resp, nil)
	}
	// 记录日志
	str, _ := json.Marshal(params)
	err := errors.New(formatRespError(resp.RespCode, resp.Error))
	bll.SendLogError(fmt.Sprintf("[SendRequestWithTimeout To %s.%s] Timeout=%d InParams=%s", module, route, timeout, string(str)), err)
	return nil, err
}

// SendNotice 发送通知
func (bll *Service) SendNotice(route string, content any) {
	err := bll.adapter.SendNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("[SendNotice To %s] InParams=%s", route, string(str)), err)
	}
	bll.sendCallback(easyCon.EPTypeNotice, route, content)
}

// SendRetainNotice 发送保持通知
func (bll *Service) SendRetainNotice(route string, content any) {
	err := bll.adapter.SendRetainNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("[SendRetainNotice To %s] InParams=%s", route, string(str)), err)
	}
	bll.sendCallback(easyCon.EPTypeNotice, route, content)
}

// SendLogDebug 发送Debug日志
func (bll *Service) SendLogDebug(content string) {
	fmt.Println(fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), content))
	bll.adapter.Debug(content)
	bll.sendCallback(easyCon.EPTypeLog, "Debug", content)
}

// SendLogWarn 发送Warn日志
func (bll *Service) SendLogWarn(content string) {
	bll.adapter.Warn(content)
	bll.sendCallback(easyCon.EPTypeLog, "Warn", content)
}

// SendLogError 发送Error日志
func (bll *Service) SendLogError(content string, err error) {
	// 记录日志
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	writeLog(bll.cfg.getBase().module, "Error", content, errStr)

	bll.adapter.Err(content, err)
	bll.sendCallback(easyCon.EPTypeLog, "Error", fmt.Sprintf("%s %s", content, errStr))
}

func (bll *Service) sendCallback(pType easyCon.EPType, route string, content any) {
	if bll.callback == nil {
		return
	}
	ctx := ""
	if v, ok := content.(string); ok == true {
		ctx = v
	} else {
		j, _ := json.Marshal(content)
		ctx = string(j)
	}

	req := CallbackReq{
		PType:   pType,
		ReqTime: qtime.NewDateTime(time.Now()).ToString(),
		Route:   route,
		Content: ctx,
	}
	// 调用回调
	reqJson, _ := json.Marshal(req)
	bll.callback(string(reqJson))
}

func (bll *Service) config() IConfig {
	return bll.cfg
}

func (bll *Service) setEnv(reg *Reg, adapter easyCon.IAdapter, callback CallbackDelegate) {
	bll.reg = reg
	bll.adapter = adapter
	bll.callback = callback
}
