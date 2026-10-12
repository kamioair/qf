package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	easyCon "github.com/qiu-tec/easy-con.golang"
	"github.com/robfig/cron/v3"
)

type IService interface {
	Reg(reg *Reg) // 注册事件

	SendLogDebug(content string) // 日志
	SendLogWarn(content string)
	SendLogError(content string, err error)

	// 内部使用的方法
	stop()
	setConfig(config *BaseConfig)
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

type OnReqFunc func(ctx *Context) (any, error)

type OnNoticeFunc func(ctx *Context)

type Service struct {
	cronList []ICron
	adapter  easyCon.IAdapter
	config   *BaseConfig
	writeLog func(level string, content string, err string)
}

func (b *Service) Invoke(pack easyCon.PackReq, onReq OnReqFunc) (easyCon.EResp, any) {
	ctx, err1 := NewContext(pack.Content, &pack, nil)
	if err1 != nil {
		return easyCon.ERespBadReq, err1.Error()
	}
	res, err2 := onReq(ctx)
	if err2 != nil {
		return easyCon.ERespError, err2.Error()
	}
	return easyCon.ERespSuccess, res
}

func (b *Service) NoticeInvoke(pack easyCon.PackNotice, onReq OnNoticeFunc) {
	ctx, err := NewContext(pack.Content, nil, &pack)
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

// SendRequest 发送请求并按指定类型返回响应。
//
// 返回值约定：
//   - T 为 any：直接透传远端 Content（不做任何转换，省一次 JSON 往返）
//   - T 为其他类型：按 Context.Get 整体取值的方式，将远端 Content 经 JSON 往返反序列化为 T
//
// 调用方须显式指定类型实参，例如：
//
//	user, err := svc.SendRequest[User]("user", "get", params)
//	list, err := svc.SendRequest[[]User]("user", "list", params)
//	raw,  err := svc.SendRequest[any]("user", "raw", params)
//
//	@param module 目标模块
//	@param route  目标路由
//	@param params 入参（任意可序列化类型）
//	@return T      响应数据
//	@return error 业务错误 / 反序列化失败
func (bll *Service) SendRequest[T any](module, route string, params any) (T, error) {
	var zero T

	resp := bll.adapter.Req(module, route, params)
	if resp.RespCode != easyCon.ERespSuccess {
		err := errors.New(fmt.Sprintf("RespCode=%v, Msg=%s", resp.RespCode, resp.Error))
		str, _ := json.Marshal(params)
		bll.SendLogError(fmt.Sprintf("SendRequest To %s.%s Error InParams=%s", module, route, string(str)), err)
		return zero, err
	}

	// T 为 any 时直接透传，避免一次冗余的 JSON 往返
	if reflect.TypeOf(zero) == reflect.TypeOf((*any)(nil)).Elem() {
		return any(resp.Content).(T), nil
	}

	// 其他类型：与 Context.Get 整体取值一致，走 JSON 序列化-反序列化路径
	js, err := json.Marshal(resp.Content)
	if err != nil {
		return zero, errors.New(fmt.Sprintf("marshal response failed: %v", err))
	}
	if err := json.Unmarshal(js, &zero); err != nil {
		return zero, errors.New(fmt.Sprintf("unmarshal response to %T failed: %v", zero, err))
	}
	return zero, nil
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
	bll.writeLog("Error", content, err.Error())
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

func (bll *Service) setConfig(config *BaseConfig) {
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
