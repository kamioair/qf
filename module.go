package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"sync"
	"time"
)

// NewModule 创建Cmd模块
func NewModule(name, desc, version string, service IService, config IConfig) IModule {
	return newModule(name, desc, version, service, config, nil)
}

// NewDllModule 创建Dll模块
func NewDllModule(name, desc, version string, service IService, config IConfig, callback CallbackDelegate) IModule {
	return newModule(name, desc, version, service, config, callback)
}

type module struct {
	service         IService
	reg             *Reg
	config          *Config
	adapter         easyCon.IAdapter
	waitConnectChan chan bool
	waitLock        sync.Mutex
	callback        CallbackDelegate
}

func newModule(name, desc, version string, service IService, config IConfig, callback CallbackDelegate) IModule {
	if service == nil {
		panic(errors.New("service cannot be nil"))
	}

	// 创建基础模块
	m := &module{
		service:         service,
		waitConnectChan: make(chan bool),
		waitLock:        sync.Mutex{},
		callback:        callback,
	}

	// 注册方法
	m.reg = &Reg{}
	m.service.Reg(m.reg)

	// 加载配置
	m.config = loadConfig(name, desc, version, config)
	return m
}

// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
func (m *module) Run() any {
	if m.callback != nil {
		// dll模式
		m.start()
	} else {
		// cmd模式
		qlauncher.Run(m.start, m.stop, false)
	}
	return true
}

// Stop 停止模块
func (m *module) Stop() {
	if m.callback != nil {
		// 异步允许，则直接退出
		m.stop()
	} else {
		qlauncher.Exit()
	}
}

func (m *module) start() {
	defer errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	}, m.config.module, "init", nil)

	m.waitConnectChan = make(chan bool)
	m.waitLock = sync.Mutex{}

	cfg := m.config

	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")

	m.reg = &Reg{}
	m.service.Reg(m.reg)

	// 解密连接配置
	addr := cfg.Broker.Addr
	uid := cfg.Broker.UId
	pwd := cfg.Broker.Pwd
	if cfg.crypto != nil {
		nAddr, e := cfg.crypto.Decrypt(addr)
		if e == nil {
			addr = nAddr
		}
		nUid, e := cfg.crypto.Decrypt(uid)
		if e == nil {
			uid = nUid
		}
		nPwd, e := cfg.crypto.Decrypt(pwd)
		if e == nil {
			pwd = nPwd
		}
	}

	fmt.Printf("Connecting Broker... (Addr: %s) ", addr)
	// 创建easyCon客户端
	setting := easyCon.NewDefaultMqttSetting(cfg.module, addr)
	setting.UID = uid
	setting.PWD = pwd
	setting.TimeOut = time.Duration(cfg.Broker.TimeOut) * time.Millisecond
	setting.ReTry = cfg.Broker.Retry
	setting.LogMode = easyCon.ELogMode(cfg.Broker.LogMode)
	setting.PreFix = cfg.Broker.Prefix
	setting.IsWaitLink = cfg.Broker.LinkTimeOut == 0
	callback := easyCon.AdapterCallBack{
		OnStatusChanged: m.onState,
		OnReqRec:        m.onReq,
		OnRespRec:       nil,
		OnExiting:       m.onExiting,
		OnGetVersion:    m.onGetVersion,
	}
	if m.reg.OnNotice != nil {
		callback.OnNoticeRec = m.reg.OnNotice
	}
	if m.reg.OnRetainNotice != nil {
		callback.OnRetainNoticeRec = m.reg.OnRetainNotice
	}
	if m.reg.OnLog != nil {
		callback.OnLogRec = m.reg.OnLog
	}
	// 创建模块链接
	m.adapter = easyCon.NewMqttAdapter(setting, callback)
	m.service.setEnv(m.reg, m.adapter, m.config, m.callback)

	// 等待连接成功
	time.Sleep(time.Millisecond * 1)
	if cfg.Broker.LinkTimeOut > 0 {
		select {
		case <-m.waitConnectChan:
			fmt.Printf("[Link]")
			break
		case <-time.After(time.Duration(cfg.Broker.LinkTimeOut) * time.Millisecond):
			// 连接超时，也继续
			fmt.Printf("[Wait]")
			break
		}
	}

	// 调用业务的初始化
	if m.reg.OnInit != nil {
		m.reg.OnInit()
	}

	// 保存配置文件
	saveConfigFile(cfg)

	// 启动成功
	fmt.Printf("\nStart OK\n\n")
}

func (m *module) stop() {
	// 调用业务的退出
	if m.reg.OnStop != nil {
		m.reg.OnStop()
	}
	// 退出客户端
	if m.adapter != nil {
		m.adapter.Stop()
	}
	fmt.Println("Module stop ok")
}

func (m *module) onExiting() {
	qlauncher.Exit()
}

func (m *module) onState(status easyCon.EStatus) {
	m.waitLock.Lock()
	defer m.waitLock.Unlock()
	fmt.Printf("Client link state = [%s]\n", status)

	if status == easyCon.EStatusLinked {
		ch := m.waitConnectChan
		m.waitConnectChan = nil // 先清空，再解锁
		// 在锁外进行channel操作
		if ch != nil {
			select {
			case ch <- true: // 非阻塞发送
				close(ch)
			default: // 防止阻塞
				close(ch)
			}
		}
	}
	if m.reg != nil && m.reg.OnStatusChanged != nil {
		go m.reg.OnStatusChanged(status)
	}
}

func (m *module) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = errors.New(err)
	}, m.config.module, pack.Route, pack.Content)

	switch pack.Route {
	case "Exit":
		m.Stop()
		return easyCon.ERespSuccess, nil
	case "Version":
		ver := map[string]string{}
		cfg := m.config
		ver["Module"] = cfg.module
		ver["Desc"] = cfg.desc
		ver["ModuleVersion"] = cfg.version
		ver["FrameVersion"] = Version
		return easyCon.ERespSuccess, ver
	}
	if m.reg.OnReq != nil {
		code, resp = m.reg.OnReq(pack)
		if code != easyCon.ERespSuccess {
			// 记录日志
			str, _ := json.Marshal(pack.Content)
			errStr := ""
			if e := resp.(error); e != nil {
				errStr = e.Error()
			} else {
				errStr = fmt.Sprintf("%v", resp)
			}
			writeLog(m.config.module, "Error", fmt.Sprintf("[OnReq From %s.%s] InParam=%s", pack.From, pack.Route, str), formatRespError(code, errStr))
		}
		return code, resp
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (m *module) onGetVersion() []string {
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", m.config.version)}
}
