package qf

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/utils/qconfig"
	"github.com/kamioair/utils/qio"
	"os"
)

type IConfig any

type BaseConfig struct {
	module   string // 模块服务名称
	desc     string // 模块服务描述
	version  string // 模块服务版本
	filePath string // 配置文件路径
	exit     string // 检查进程退出
	Broker   struct {
		Addr    string // 地址
		UId     string // 用户名
		Pwd     string // 密码
		TimeOut int    // 连接超时
		Retry   int    // 重试次数
		LogMode string // 日志模式
		Prefix  string // 前缀
	} `comment:"MqBroker\n Addr:访问地址\n UId,Pwd:登录账号密码\n TimeOut:超时(毫秒)\n Retry:重试次数\n LogMode:日志模式 NONE/CONSOLE\n Prefix:前缀"` // 服务连接配置
}

// loadConfig 加载配置文件
func loadConfig(name, desc, version string, config IConfig) *BaseConfig {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	opts := qconfig.SaveContent{}

	// 加载基础配置
	baseCfg := initBaseConfig(name, desc, version)
	err = qconfig.LoadConfig(baseCfg.filePath, "Base", baseCfg)
	if err != nil {
		panic(err)
	}
	opts.Add("Base", "Base BaseConfig", baseCfg)

	// 加载模块自定义配置
	if config != nil {
		err = qconfig.LoadConfig(baseCfg.filePath, name, config)
		if err != nil {
			panic(err)
		}
		opts.Add(name, desc, config)
	}

	// 保存配置
	err = qconfig.SaveConfig(baseCfg.filePath, opts)
	if err != nil {
		fmt.Printf("保存配置文件失败: %v\n", err)
	}

	// 如果有外部传入参数，则更新配置
	setByArgs(baseCfg)

	return baseCfg
}

// initBaseConfig 初始化基础默认配置
func initBaseConfig(name, desc, version string) *BaseConfig {
	cfg := &BaseConfig{}
	cfg.module = name
	cfg.desc = desc
	cfg.version = version
	cfg.filePath = "./config.yaml"
	cfg.Broker = struct {
		Addr    string // 地址
		UId     string // 用户名
		Pwd     string // 密码
		TimeOut int    // 连接超时
		Retry   int    // 重试次数
		LogMode string // 日志模式
		Prefix  string // 前缀
	}{
		Addr:    "ws://127.0.0.1:5002/ws",
		UId:     "",
		Pwd:     "",
		TimeOut: 3000,
		Retry:   3,
		LogMode: "NONE",
	}
	return cfg
}

// setByArgs 根据外部传参更新基础配置
func setByArgs(config *BaseConfig) {
	// 如果有入参，则用入参（仅处理ConfigPath，其他参数在loadConfig中处理）
	if len(os.Args) > 1 {
		args := map[string]string{}
		err := json.Unmarshal([]byte(os.Args[1]), &args)
		if err != nil {
			return
		}
		// 自定义配置文件路径
		if val, ok := args["ConfigPath"]; ok {
			config.filePath = val
		}
		// 自定义模块名称
		if val, ok := args["Module"]; ok && val != "" {
			config.module = val
		}
		// 自定义Broker配置
		if val, ok := args["Broker"]; ok {
			err = json.Unmarshal([]byte(val), &config.Broker)
			if err != nil {
				panic(err)
			}
		}
	}
}
