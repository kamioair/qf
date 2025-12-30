package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type plugin struct {
	service IService
	reg     *Reg
	config  *Config
	adapter easyCon.IAdapter
	onWrite OnWriteDelegate
	onRead  OnReadDelegate
}

func NewPlugin(name, desc, version string, service IService, config IConfig, onWrite OnWriteDelegate) IModule {
	p := &plugin{
		service: service,
		onWrite: onWrite,
	}

	// 注册方法
	p.reg = &Reg{}
	p.service.Reg(p.reg)

	// 加载配置
	p.config = loadConfig(name, desc, version, config)

	return p
}

func (p *plugin) Run() any {
	defer errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	}, p.config.module, "init", nil)

	cfg := p.config

	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")

	setting := easyCon.CoreSetting{
		Module:            cfg.module,
		TimeOut:           time.Duration(cfg.Broker.TimeOut) * time.Millisecond,
		ReTry:             cfg.Broker.Retry,
		LogMode:           easyCon.ELogMode(cfg.Broker.LogMode),
		PreFix:            cfg.Broker.Prefix,
		ChannelBufferSize: 100,
		ConnectRetryDelay: time.Second,
		IsWaitLink:        cfg.Broker.LinkTimeOut == 0,
		IsSync:            false,
	}

	callback := easyCon.AdapterCallBack{
		OnStatusChanged: p.onState,
		OnReqRec:        p.onReq,
		OnRespRec:       nil,
		OnExiting:       p.onExiting,
		OnGetVersion:    p.onGetVersion,
	}
	if p.reg.OnNotice != nil {
		callback.OnNoticeRec = p.reg.OnNotice
	}
	if p.reg.OnRetainNotice != nil {
		callback.OnRetainNoticeRec = p.reg.OnRetainNotice
	}
	if p.reg.OnLog != nil {
		callback.OnLogRec = p.reg.OnLog
	}

	p.adapter, p.onRead = easyCon.NewCgoAdapter(setting, callback, p.onWrite)

	// 调用业务的初始化
	p.service.setEnv(p.reg, p.adapter, p.config, nil)
	if p.reg.OnInit != nil {
		p.reg.OnInit()
	}

	// 保存配置文件
	saveConfigFile(p.config)

	// 启动成功
	fmt.Printf("\nStart OK\n\n")

	return p.onRead
}

func (p *plugin) Stop() {

}

func (p *plugin) onExiting() {
	qlauncher.Exit()
}

func (p *plugin) onState(status easyCon.EStatus) {
	fmt.Printf("Client link state = [%s]\n", status)

	if p.reg != nil && p.reg.OnStatusChanged != nil {
		go p.reg.OnStatusChanged(status)
	}
}

func (p *plugin) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = errors.New(err)
	}, p.config.module, pack.Route, pack.Content)

	switch pack.Route {
	case "Exit":
		p.Stop()
		return easyCon.ERespSuccess, nil
	case "Version":
		ver := map[string]string{}
		cfg := p.config
		ver["Module"] = cfg.module
		ver["Desc"] = cfg.desc
		ver["ModuleVersion"] = cfg.version
		ver["FrameVersion"] = Version
		return easyCon.ERespSuccess, ver
	}
	if p.reg.OnReq != nil {
		code, resp = p.reg.OnReq(pack)
		if code != easyCon.ERespSuccess {
			// 记录日志
			str, _ := json.Marshal(pack.Content)
			errStr := ""
			if e := resp.(error); e != nil {
				errStr = e.Error()
			} else {
				errStr = fmt.Sprintf("%v", resp)
			}
			writeLog(p.config.module, "Error", fmt.Sprintf("[OnReq From %s.%s] InParam=%s", pack.From, pack.Route, str), formatRespError(code, errStr))
		}
		return code, resp
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (p *plugin) onGetVersion() []string {
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", p.config.version)}
}
