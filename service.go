package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qtime"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type Reg struct {
	OnInit          func()
	OnStop          func()
	OnReq           func(pack easyCon.PackReq) (easyCon.EResp, any)
	OnNotice        func(notice easyCon.PackNotice)
	OnRetainNotice  func(notice easyCon.PackNotice)
	OnStatusChanged func(status easyCon.EStatus)
	OnLog           func(log easyCon.PackLog)
}

// CallbackDelegate 回调
type CallbackDelegate func(inParam string) (string, error)

type OnReqFunc func(ctx IContext) (any, error)

type OnNoticeFunc func(ctx IContext)

type Service struct {
	adapter  easyCon.IAdapter
	config   *Config
	reg      *Reg
	callback CallbackDelegate
}

// GetInvokes 获取绑定事件
func (bll *Service) GetInvokes() *Reg {
	return bll.reg
}

// Invoke 调用请求实现方法
func (bll *Service) Invoke(pack easyCon.PackReq, onReq OnReqFunc) (code easyCon.EResp, resp any) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = errors.New(err)
		bll.SendLogError("invoke panic", errors.New(fmt.Sprintf("code = %s, error = %s", code, resp)))
	}, bll.config.module, pack.Route, pack.Content)

	// 创建上下文
	ctx, err := newContent(pack.Content, &pack, nil, nil)
	if err != nil {
		return easyCon.ERespBadReq, err
	}
	res, err := onReq(ctx)
	if err != nil {
		return easyCon.ERespError, err
	}
	return easyCon.ERespSuccess, res
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
	cfg := bll.config.CallBack.Notice
	if cfg == ECallBackAll || cfg == ECallBackUp {
		err := bll.adapter.SendNotice(route, content)
		if err != nil {
			str, _ := json.Marshal(content)
			bll.SendLogError(fmt.Sprintf("[SendNotice To %s] InParams=%s", route, string(str)), err)
		}
	}
	bll.sendCallback(easyCon.EPTypeNotice, route, content)
}

// SendRetainNotice 发送保持通知
func (bll *Service) SendRetainNotice(route string, content any) {
	cfg := bll.config.CallBack.Notice
	if cfg == ECallBackAll || cfg == ECallBackUp {
		err := bll.adapter.SendRetainNotice(route, content)
		if err != nil {
			str, _ := json.Marshal(content)
			bll.SendLogError(fmt.Sprintf("[SendRetainNotice To %s] InParams=%s", route, string(str)), err)
		}
	}
	bll.sendCallback(easyCon.EPTypeNotice, route, content)
}

// SendLogDebug 发送Debug日志
func (bll *Service) SendLogDebug(content string) {
	fmt.Println(fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), content))
	cfg := bll.config.CallBack.Notice
	if cfg == ECallBackAll || cfg == ECallBackUp {
		bll.adapter.Debug(content)
	}
	bll.sendCallback(easyCon.EPTypeLog, "Debug", content)
}

// SendLogWarn 发送Warn日志
func (bll *Service) SendLogWarn(content string) {
	cfg := bll.config.CallBack.Notice
	if cfg == ECallBackAll || cfg == ECallBackUp {
		bll.adapter.Warn(content)
	}
	bll.sendCallback(easyCon.EPTypeLog, "Warn", content)
}

// SendLogError 发送Error日志
func (bll *Service) SendLogError(content string, err error) {
	// 记录日志
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	writeLog(bll.config.module, "Error", content, errStr)

	cfg := bll.config.CallBack.Notice
	if cfg == ECallBackAll || cfg == ECallBackUp {
		bll.adapter.Err(content, err)
	}
	bll.sendCallback(easyCon.EPTypeLog, "Error", content)
}

func (bll *Service) sendCallback(pType easyCon.EPType, route string, content any) {
	cfg := bll.config.CallBack.Notice
	if (cfg == ECallBackAll || cfg == ECallBackBack) && bll.callback != nil {
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
		_, err := bll.callback(string(reqJson))
		if err != nil {
			bll.SendLogError(fmt.Sprintf("[SendCallback Error %s.%s] InParams=%s", pType, route, ctx), err)
		}
	}
}

func (bll *Service) setEnv(reg *Reg, adapter easyCon.IAdapter, config *Config, callback CallbackDelegate) {
	bll.reg = reg
	bll.adapter = adapter
	bll.config = config
	bll.callback = callback
}
