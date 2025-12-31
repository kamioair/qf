package qf

import (
	"fmt"
	"github.com/kamioair/utils/qconfig"
	"github.com/kamioair/utils/qio"
	"os"
)

type Config struct {
	module        string  // 模块服务名称
	desc          string  // 模块服务描述
	version       string  // 模块服务版本
	filePath      string  // 配置文件路径
	exit          string  // 检查进程退出
	crypto        ICrypto // 加解密接口
	customSection string  // 自定义配置节名称
	Broker        struct {
		Addr             string // 地址
		UId              string // 用户名
		Pwd              string // 密码
		TimeOut          int    // 连接超时
		Retry            int    // 重试次数
		LogMode          string // 日志模式
		Prefix           string // 前缀
		LinkTimeOut      int    // 连接等待时间
		IsRandomClientID bool   // 是否随机clientID
		IsSyncMode       bool   // 是否同步模式
	} `comment:"MqBroker\n Addr:访问地址\n UId,Pwd:登录账号密码\n TimeOut:请求超时(毫秒)\n Retry:重试次数\n LogMode:日志模式 NONE/CONSOLE\n Prefix:前缀，用于同一个模块不同实例\n LinkTimeOut:连接等待超时(毫秒) 0表示无限等待直到连上\n IsRandomClientID:是否随机clientID\n IsSyncMode:是否请求同步模式，启用后所有请求无法并行，只能一个一个执行"` // 服务连接配置
	CallBack struct {
		Notice string
		Log    string
	} `comment:"CallBack回调配置 Back/Up/All\n Notice:通知回调\n Log:日志回调"`
}

const (
	ECallBackBack = "Back"
	ECallBackUp   = "Up"
	ECallBackAll  = "All"
)

type emptyConfig struct {
	Config
}

var (
	opts = qconfig.SaveConfigOptions{
		SectionDescs: map[string]string{},
	}
)

// GetModuleInfo 获取基础配置（给外部用）
func (c *Config) GetModuleInfo() (Name string, Desc string, Version string) {
	return c.module, c.desc, c.version
}

func (c *Config) RegCrypto(crypto ICrypto) {
	c.crypto = crypto
}

func (c *Config) SetCustomSection(section string) {
	c.customSection = section
}

// getBase 获取基础配置（供内部module.go调用）
func (c *Config) getBase() *Config {
	return c
}

// setBase 设置基础配置
func (c *Config) setBase(name, desc, version string) {
	c.module = name
	c.desc = desc
	c.version = version
}

// loadConfig 加载配置文件
func loadConfig(config IConfig, customSetting map[string]any) *Config {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	if config == nil {
		config = &emptyConfig{}
	}

	// 加载基础配置
	baseCfg := config.getBase()
	baseCfg.filePath = "./config.yaml"
	baseCfg.Broker = struct {
		Addr             string // 地址
		UId              string // 用户名
		Pwd              string // 密码
		TimeOut          int    // 连接超时
		Retry            int    // 重试次数
		LogMode          string // 日志模式
		Prefix           string // 前缀
		LinkTimeOut      int    // 连接等待时间
		IsRandomClientID bool   // 是否随机clientID
		IsSyncMode       bool
	}{
		Addr:             "ws://127.0.0.1:5002/ws",
		UId:              "",
		Pwd:              "",
		TimeOut:          3000,
		Retry:            3,
		LogMode:          "NONE",
		LinkTimeOut:      3000,
		IsRandomClientID: false,
		IsSyncMode:       false,
	}
	baseCfg.CallBack = struct {
		Notice string
		Log    string
	}{
		Notice: "All",
		Log:    "All",
	}
	// 如果有外部传入参数，则更新配置
	if customSetting != nil {
		// 自定义配置文件路径
		if val, ok := customSetting["ConfigPath"]; ok {
			baseCfg.filePath = val.(string)
		}
		// 自定义模块名称
		if val, ok := customSetting["Module"]; ok && val != "" {
			baseCfg.module = val.(string)
		}
	}
	fileExist := qio.PathExists(baseCfg.filePath)
	err = qconfig.LoadConfig(baseCfg.filePath, "Base", baseCfg)
	if err != nil {
		panic(err)
	}

	// 加载模块自定义配置
	section := baseCfg.customSection
	if section == "" {
		section = baseCfg.module
	}
	err = qconfig.LoadConfig(baseCfg.filePath, section, config)
	if err != nil {
		panic(err)
	}

	// 首次创建配置文件，立即保存
	if fileExist == false {
		saveConfigFile(baseCfg)
	}

	return config.getBase()
}

// saveConfigFile 保存配置文件（供内部module.go调用）
func saveConfigFile(config *Config) {
	// 准备保存选项
	section := config.customSection
	if section == "" {
		section = config.module
	}
	opts.SectionDescs[section] = config.desc

	// 保存配置
	err := qconfig.SaveConfig(config.filePath, &opts)
	if err != nil {
		fmt.Printf("保存配置文件失败: %v\n", err)
	}
}
