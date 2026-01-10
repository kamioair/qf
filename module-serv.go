package qf

import (
	"fmt"
	"github.com/kamioair/utils/qconvert"
	"github.com/kamioair/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"sync"
	"time"
)

// NewModule 创建Cmd模块
func NewModule(service IService) IModule {
	return newModule(service, nil)
}

// NewDllModule 创建Dll模块
func NewDllModule(service IService, callback CallbackDelegate) IModule {
	return newModule(service, callback)
}

type module struct {
	*baseModule     // 嵌入基础模块
	waitConnectChan chan bool
	waitLock        sync.Mutex
	callback        CallbackDelegate
}

func newModule(service IService, callback CallbackDelegate) IModule {
	return &module{
		baseModule:      newBaseModule(service),
		callback:        callback,
		waitConnectChan: make(chan bool),
		waitLock:        sync.Mutex{},
	}
}

// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
func (m *module) Run() {
	// cmd模式
	qlauncher.Run(m.start, m.stop, false)
}

// RunAsync 异步运行模块，执行后不等待
func (m *module) RunAsync() {
	// dll模式
	m.start()
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
	cfg := m.service.config().getBase()

	defer errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	}, cfg.module, "init", nil)

	m.waitConnectChan = make(chan bool)
	m.waitLock = sync.Mutex{}

	// 打印模块信息
	m.printModuleInfo()

	// 重新注册（确保初始化）
	m.reg = &Reg{}
	m.service.Reg(m.reg)

	// 解密连接配置
	addr, uid, pwd := m.decryptBrokerConfig()

	fmt.Printf("Connecting Broker... (Addr: %s) ", addr)
	name := cfg.module
	if cfg.Broker.IsRandomClientID {
		name = fmt.Sprintf("%s-%s", name, qconvert.Time.ToString(time.Now(), "yyyyMMddHHmmssfff"))
	}

	// 创建easyCon客户端
	setting := easyCon.NewDefaultMqttSetting(name, addr)
	setting.UID = uid
	setting.PWD = pwd
	setting.TimeOut = time.Duration(cfg.Broker.TimeOut) * time.Millisecond
	setting.ReTry = cfg.Broker.Retry
	setting.LogMode = easyCon.ELogMode(cfg.Broker.LogMode)
	setting.PreFix = cfg.Broker.Prefix
	setting.IsWaitLink = cfg.Broker.LinkTimeOut == 0
	setting.IsSync = cfg.Broker.IsSyncMode

	// 构建回调
	callback := m.buildAdapterCallBack(m.onState, m.onReq, m.onExiting, m.getVersion)

	// 创建模块链接
	m.adapter = easyCon.NewMqttAdapter(setting, callback)
	m.setEnv(m.callback)

	// 等待连接成功
	time.Sleep(time.Millisecond * 1)
	if cfg.Broker.LinkTimeOut > 0 {
		select {
		case <-m.waitConnectChan:
			break
		case <-time.After(time.Duration(cfg.Broker.LinkTimeOut) * time.Millisecond):
			// 连接超时，也继续
			fmt.Printf("Link state = [BackConn]\n")
			break
		}
	}

	// 调用业务的初始化
	m.callOnInit()

	// 保存配置文件
	m.saveConfig()

	// 启动成功
	fmt.Printf("\nStart OK\n\n")
}

func (m *module) stop() {
	// 调用业务的退出
	m.callOnStop()
	// 退出客户端
	m.stopAdapter()
	fmt.Println("Module stop ok")
}

func (m *module) onExiting() {
	qlauncher.Exit()
}

func (m *module) onState(status easyCon.EStatus) {

	if status == easyCon.EStatusLinked {
		m.waitLock.Lock()
		defer m.waitLock.Unlock()

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

	m.callOnState(status)
}

func (m *module) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	return m.handleReq(pack, m.Stop)
}
