package qf

import (
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// IModule 模块入口接口
type IModule interface {
	// Run 运行模块
	Run()
	// Stop 停止模块
	Stop()
}

// IService 模块功能接口
type IService interface {
	Reg(reg *Reg)                                                                  // 注册事件
	GetInvokes() *Reg                                                              // 返回注册事件
	Load(name, desc, version string, config IConfig, customSetting map[string]any) // 加载模块

	// 内部使用的方法
	config() IConfig
	setEnv(reg *Reg, adapter easyCon.IAdapter, callback CallbackDelegate)
}

// IConfig 配置接口
type IConfig interface {
	setBase(name, desc, version string)
	getBase() *Config
}

// ICrypto 加解密接口
type ICrypto interface {
	// Decrypt 解密
	Decrypt(content string) (string, error)
}

// IContext 上下文
type IContext interface {
	Raw() string
	Bind(refStruct any) error
}

// Reg 事件绑定
type Reg struct {
	OnInit          func()
	OnStop          func()
	OnReq           func(pack easyCon.PackReq) (easyCon.EResp, any)
	OnNotice        func(notice easyCon.PackNotice)
	OnRetainNotice  func(notice easyCon.PackNotice)
	OnStatusChanged func(status easyCon.EStatus)
	OnLog           func(log easyCon.PackLog)
}

// OnReqFunc 请求方法定义
type OnReqFunc func(ctx IContext) (any, error)

// OnNoticeFunc 通知方法定义
type OnNoticeFunc func(ctx IContext)

// CallbackDelegate 回调
type CallbackDelegate func(inParam string)

// OnWriteDelegate 插件用委托
type OnWriteDelegate func([]byte) error
type OnReadDelegate func([]byte)

type CallbackReq struct {
	PType   easyCon.EPType
	ReqTime string
	Route   string
	Content string
}

const (
	runModeCmd = "cmd"
	runModeDll = "dll"
)
