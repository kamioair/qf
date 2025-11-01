package qf

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/utils/qconfig"
	"github.com/kamioair/utils/qio"
	"os"
	"strings"
)

type IConfig interface {
	getBaseConfig() *Config
}

type Config struct {
	module   string // 模块服务名称
	desc     string // 模块服务描述
	version  string // 模块服务版本
	filePath string // 配置文件路径
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

var baseCfg *Config

func (c *Config) getBaseConfig() *Config {
	return c
}

func (c *Config) GetModuleInfo() (Name string, Desc string, Version string) {
	return c.module, c.desc, c.version
}

func loadConfig(name, desc, version string, config IConfig) {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	// 加载初始值
	baseCfg = initBaseConfig(name, desc, version, config)

	// 使用统一的配置加载方法
	configs := map[string]interface{}{
		"Base": baseCfg,
		name:   config,
	}

	// 加载配置
	err = qconfig.LoadConfig(baseCfg.filePath, configs)
	if err != nil {
		panic(err)
	}

	// 生成配置内容字符串
	configBase := map[string]any{}
	configBase["Base"] = baseCfg
	configModule := map[string]any{}
	configModule[baseCfg.module] = config
	newCfg := ""
	newCfg += "############################### Base Config ###############################\n"
	newCfg += "# 通用基础配置\n"
	newCfg += qconfig.ToYAML(configBase, 0, []string{})

	mCfg := fmt.Sprintf("############################### %s Config ###############################\n", baseCfg.module)
	mCfg += fmt.Sprintf("# %s\n", baseCfg.desc)
	mCfg += qconfig.ToYAML(configModule, 0, []string{"Config"})
	if strings.HasSuffix(mCfg, fmt.Sprintf("%s: \n", baseCfg.module)) == false {
		newCfg += "\n\n"
		newCfg += mCfg
	}

	// 尝试检测是否有变化，如果有则更新文件
	qconfig.TrySave(baseCfg.filePath, newCfg)
}

func initBaseConfig(name, desc, version string, c IConfig) *Config {
	config := c.getBaseConfig()
	config.module = name
	config.desc = desc
	config.version = version
	config.filePath = "./config.yaml"
	config.Broker = struct {
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

	// 如果有入参，则用入参（仅处理ConfigPath，其他参数在loadConfig中处理）
	if len(os.Args) > 1 {
		args := map[string]string{}
		err := json.Unmarshal([]byte(os.Args[1]), &args)
		if err != nil {
			panic(err)
		}
		// 自定义配置文件路径
		if val, ok := args["ConfigPath"]; ok {
			config.filePath = val
		}
		// 自定义模块名称
		if val, ok := args["Module"]; ok && val != "" {
			baseCfg.module = val
		}
		// 自定义Broker配置
		if val, ok := args["Broker"]; ok {
			err = json.Unmarshal([]byte(val), &baseCfg.Broker)
			if err != nil {
				panic(err)
			}
		}
	}

	return config
}
