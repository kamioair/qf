package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"net"
	"time"
)

type IService interface {
	Reg(reg *Reg) // 注册事件

	SendLogDebug(content string) // 日志
	SendLogWarn(content string)
	SendLogError(content string, err error)

	// 内部使用的方法
	stop()
	setConfig(config *Config)
	setAdapter(adapter easyCon.IAdapter)
	setWriteLog(writeLog func(level string, content string, err string))
}

type Reg struct {
	OnInit          func()
	OnStop          func()
	OnReq           func(pack easyCon.PackReq) (easyCon.EResp, any)
	OnNotice        func(notice easyCon.PackNotice)
	OnRetainNotice  func(notice easyCon.PackNotice)
	OnStatusChanged func(status easyCon.EStatus)
	OnLog           func(log easyCon.PackLog)
}

type OnReqFunc func(ctx IContext) (any, error)

type OnNoticeFunc func(ctx IContext)

type Service struct {
	adapter  easyCon.IAdapter
	config   *Config
	writeLog func(level string, content string, err string)
}

// Invoke 调用请求实现方法
func (bll *Service) Invoke(pack easyCon.PackReq, onReq OnReqFunc) (easyCon.EResp, any) {
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
	err := bll.adapter.SendNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("[SendNotice To %s] InParams=%s", route, string(str)), err)
	}
}

// SendRetainNotice 发送保持通知
func (bll *Service) SendRetainNotice(route string, content any) {
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
	bll.writeLog("Debug", content, "")
}

// SendLogWarn 发送Warn日志
func (bll *Service) SendLogWarn(content string) {
	bll.adapter.Warn(content)
	bll.writeLog("Warn", content, "")
}

// SendLogError 发送Error日志
func (bll *Service) SendLogError(content string, err error) {
	bll.adapter.Err(content, err)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	bll.writeLog("Error", content, errStr)
}

func (bll *Service) stop() {

}

func (bll *Service) setAdapter(adapter easyCon.IAdapter) {
	bll.adapter = adapter
}

func (bll *Service) setWriteLog(writeLog func(level string, content string, err string)) {
	bll.writeLog = writeLog
}

func (bll *Service) setConfig(config *Config) {
	bll.config = config
}

func (bll *Service) getIp() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	ips := []string{}
	for _, face := range interfaces {
		// 跳过未启用的接口或环回接口
		if face.Flags&net.FlagUp == 0 || face.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取该接口的地址列表
		addrList, err := face.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrList {
			// 检查地址是否为 IP 地址
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}

	str, _ := json.Marshal(ips)
	return string(str)
}
