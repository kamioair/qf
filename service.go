package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type Service struct {
	adapter easyCon.IAdapter
	cfg     IConfig
	reg     *Reg
}

// GetRegEvents 获取注册绑定事件
func (bll *Service) GetRegEvents() *Reg {
	return bll.reg
}

// Name 返回模块名称
func (bll *Service) Name() string {
	return bll.cfg.getBase().module
}

// Load 初始化
func (bll *Service) Load(moduleName, moduleDesc, moduleVersion string, customSectionName string, config IConfig) {
	bll.cfg = config
	if bll.cfg == nil {
		bll.cfg = &emptyConfig{}
	}
	// 设置模块信息
	bll.cfg.setBase(moduleName, moduleDesc, moduleVersion, customSectionName)
	// 加载配置
	loadConfig(bll.cfg)
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
func (bll *Service) ReturnOk(content []byte) (code easyCon.EResp, resp []byte) {
	return easyCon.ERespSuccess, content
}

// ReturnErr 返回错误
func (bll *Service) ReturnErr(content []byte) (code easyCon.EResp, resp []byte) {
	return easyCon.ERespError, content
}

// ReturnNotFind 返回未找到
func (bll *Service) ReturnNotFind() (code easyCon.EResp, resp []byte) {
	return easyCon.ERespRouteNotFind, nil
}

// SendRequest 发送请求
func (bll *Service) SendRequest(module, route string, params []byte) easyCon.PackResp {
	resp := bll.adapter.Req(module, route, params)
	if resp.RespCode != easyCon.ERespSuccess {
		// 记录日志
		str, _ := json.Marshal(params)
		err := errors.New(formatRespError(resp.RespCode, string(resp.Content)))
		bll.SendLogError(fmt.Sprintf("[SendRequest To %s.%s] InParams=%s", module, route, string(str)), err)
	}
	return resp
}

// SendRequestWithTimeout 发送请求(可自定义超时时间的,单位毫秒)
func (bll *Service) SendRequestWithTimeout(module, route string, params []byte, timeout int) easyCon.PackResp {
	resp := bll.adapter.ReqWithTimeout(module, route, params, timeout)
	if resp.RespCode != easyCon.ERespSuccess {
		// 记录日志
		str, _ := json.Marshal(params)
		err := errors.New(formatRespError(resp.RespCode, string(resp.Content)))
		bll.SendLogError(fmt.Sprintf("[SendRequestWithTimeout To %s.%s] InParams=%s", module, route, string(str)), err)
	}
	return resp
}

// SendNotice 发送通知
func (bll *Service) SendNotice(route string, content []byte) {
	err := bll.adapter.SendNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("[SendNotice To %s] InParams=%s", route, string(str)), err)
	}
}

// SendRetainNotice 发送保持通知
func (bll *Service) SendRetainNotice(route string, content []byte) {
	err := bll.adapter.SendRetainNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("[SendRetainNotice To %s] InParams=%s", route, string(str)), err)
	}
}

// SendLogDebug 发送Debug日志
func (bll *Service) SendLogDebug(content string) {
	fmt.Println(fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), content))
	bll.adapter.Debug(content)
}

// SendLogWarn 发送Warn日志
func (bll *Service) SendLogWarn(content string) {
	bll.adapter.Warn(content)
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
}

func (bll *Service) config() IConfig {
	return bll.cfg
}

func (bll *Service) setEnv(reg *Reg, adapter easyCon.IAdapter) {
	bll.reg = reg
	bll.adapter = adapter
}
