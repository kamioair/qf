package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"github.com/robfig/cron/v3"
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

type ICron interface {
	stop()
	Add(cron string, mission func()) (missionId int)
	Remove(missionId int)
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
	cronList []ICron
	adapter  easyCon.IAdapter
	config   *Config
	writeLog func(level string, content string, err string)
}

func (b *Service) Invoke(pack easyCon.PackReq, onReq OnReqFunc) (easyCon.EResp, any) {
	ctx, err := NewContent(pack.Content, &pack, nil)
	if err != nil {
		return easyCon.ERespBadReq, err
	}
	res, err := onReq(ctx)
	if err != nil {
		return easyCon.ERespError, err
	}
	return easyCon.ERespSuccess, res
}

func (b *Service) NoticeInvoke(pack easyCon.PackNotice, onReq OnNoticeFunc) {
	ctx, err := NewContent(pack.Content, nil, &pack)
	if err != nil {
		b.SendLogError(fmt.Sprintln("NoticeInvoke build invoke error", pack), err)
	}
	onReq(ctx)
}

func (bll *Service) ReturnOk(content any) (code easyCon.EResp, resp any) {
	return easyCon.ERespSuccess, content
}

func (bll *Service) ReturnErr(content any) (code easyCon.EResp, resp any) {
	js, _ := json.Marshal(content)
	return easyCon.ERespError, errors.New(string(js))
}

func (bll *Service) ReturnNotFind() (code easyCon.EResp, resp any) {
	return easyCon.ERespRouteNotFind, nil
}

// CreateCron 创建定时任务
func (bll *Service) CreateCron() ICron {
	crn := createCron()
	if bll.cronList == nil {
		bll.cronList = make([]ICron, 0)
	}
	bll.cronList = append(bll.cronList, crn)
	return crn
}

// SendRequest 发送请求
func (bll *Service) SendRequest(module, route string, params any) (IContext, error) {
	resp := bll.adapter.Req(module, route, params)
	if resp.RespCode == easyCon.ERespSuccess {
		return NewContent(resp.Content, &resp.PackReq, nil)
	}
	err := errors.New(fmt.Sprintf("%d %s %s", resp.RespCode, resp.Content, resp.Error))
	// 记录日志
	str, _ := json.Marshal(params)
	bll.SendLogError(fmt.Sprintf("SendRequest To %s.%s Error InParams=%s", module, route, string(str)), err)
	return nil, err
}

// SendNotice 发送通知
func (bll *Service) SendNotice(route string, content any) {
	err := bll.adapter.SendNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("SendNotice To %s Error InParams=%s", route, string(str)), err)
	}
}

// SendRetainNotice 发送保持通知
func (bll *Service) SendRetainNotice(route string, content any) {
	err := bll.adapter.SendRetainNotice(route, content)
	if err != nil {
		str, _ := json.Marshal(content)
		bll.SendLogError(fmt.Sprintf("SendRetainNotice To %s Error InParams=%s", route, string(str)), err)
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
	bll.writeLog("Error", content, "")
}

func (bll *Service) stop() {
	if bll.cronList != nil {
		for _, c := range bll.cronList {
			c.stop()
		}
	}
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

type cronStruct struct {
	crn *cron.Cron
}

// createCron 创建定时任务
func createCron() ICron {
	crn := &cronStruct{
		crn: cron.New(cron.WithSeconds()),
	}
	crn.crn.Start()
	return crn
}

// Stop 停止定时任务
func (crn *cronStruct) stop() {
	crn.crn.Stop()
}

// Add 添加定时方法
func (crn *cronStruct) Add(cron string, mission func()) (missionId int) {

	iid, err := crn.crn.AddFunc(cron, mission)
	if err != nil {
		panic(err)
	}
	return int(iid)
}

// Remove 移除定时方法
func (crn *cronStruct) Remove(missionId int) {
	crn.crn.Remove(cron.EntryID(missionId))
}
