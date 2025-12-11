package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type IModule interface {
	// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
	Run()
	// RunAsync 异步运行模块，执行后不等待，单进程需要启动多模块时使用（dll模式）
	RunAsync()
	// Stop 停止模块
	Stop()
}

// NewModule 创建模块
func NewModule(name, desc, version string, service IService, config IConfig) IModule {
	if service == nil {
		panic(errors.New("service cannot be nil"))
	}

	// 加载配置
	if config == nil {
		config = &emptyConfig{}
	}
	loadConfig(name, desc, version, config)

	// 创建基础模块
	m := &module{
		service: service,
		config:  config.getBaseConfig(),
	}
	instance = service
	return m
}

type module struct {
	service         IService
	reg             *Reg
	config          *Config
	adapter         easyCon.IAdapter
	waitConnectChan chan bool
	asyncRun        bool
}

// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
func (m *module) Run() {
	qlauncher.Run(m.start, m.stop, false)
}

// RunAsync 异步运行模块，执行后不等待，单进程需要启动多模块时使用（dll模式）
func (m *module) RunAsync() {
	m.asyncRun = true
	m.start()
}

// Stop 停止模块
func (m *module) Stop() {
	if m.asyncRun {
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

	cfg := m.config

	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")

	m.reg = &Reg{}
	m.service.Reg(m.reg)

	fmt.Printf("Connecting Broker... (Addr: %s) ", cfg.Broker.Addr)
	// 创建easyCon客户端
	clientId := cfg.module
	setting := easyCon.NewSetting(clientId, cfg.Broker.Addr, m.onReq, m.onState)
	if m.reg.OnNotice != nil {
		setting.OnNotice = m.reg.OnNotice
	}
	if m.reg.OnRetainNotice != nil {
		setting.OnRetainNotice = m.reg.OnRetainNotice
	}
	setting.UID = cfg.Broker.UId
	setting.PWD = cfg.Broker.Pwd
	setting.TimeOut = time.Duration(cfg.Broker.TimeOut) * time.Millisecond
	setting.ReTry = cfg.Broker.Retry
	setting.LogMode = easyCon.ELogMode(cfg.Broker.LogMode)
	setting.PreFix = cfg.Broker.Prefix
	setting.OnExiting = m.onExiting
	setting.OnGetVersion = m.onGetVersion
	setting.IsRandomClientID = cfg.Broker.IsRandomClientID
	setting.IsWaitLink = cfg.Broker.LinkTimeOut == 0
	if cfg.Broker.IsSyncMode {
		setting.EProtocol = easyCon.EProtocolMQTTSync
	}
	if m.reg.OnLog != nil {
		setting.OnLog = m.reg.OnLog
	}
	// 创建模块链接
	m.adapter = easyCon.NewMqttAdapter(setting)
	m.service.setEnv(m.reg, m.adapter, m.config)

	// 等待连接成功
	time.Sleep(time.Millisecond * 1)
	if cfg.Broker.LinkTimeOut > 0 {
		select {
		case <-m.waitConnectChan:
			fmt.Printf("[Link]")
			break
		case <-time.After(time.Duration(cfg.Broker.LinkTimeOut) * time.Millisecond):
			// 连接超时，也继续
			fmt.Printf("[UnLink]")
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

func (m *module) onState(status easyCon.EStatus) {
	if status == easyCon.EStatusLinked {
		// 连接成功
		m.waitConnectChan <- true
		close(m.waitConnectChan)
	}
	if m.reg.OnStatusChanged != nil {
		m.reg.OnStatusChanged(status)
	}
}

func (m *module) onGetVersion() []string {
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", m.config.version)}
}
