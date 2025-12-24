package qf

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/utils/qconfig"
	"github.com/kamioair/utils/qio"
	"os"
	"regexp"
	"strconv"
	"strings"
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

// getBaseConfig 获取基础配置（供内部module.go调用）
func (c *Config) getBaseConfig() *Config {
	return c
}

// loadConfig 加载配置文件
func loadConfig(name, desc, version string, config IConfig) {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	// 加载基础配置
	baseCfg := initBaseConfig(name, desc, version, config)
	fileExist := qio.PathExists(baseCfg.filePath)
	err = qconfig.LoadConfig(baseCfg.filePath, "Base", baseCfg)
	if err != nil {
		panic(err)
	}
	// 如果有外部传入参数，则更新配置
	setByArgs(baseCfg)

	// 加载模块自定义配置
	section := baseCfg.customSection
	if section == "" {
		section = name
	}
	err = qconfig.LoadConfig(baseCfg.filePath, section, config)
	if err != nil {
		panic(err)
	}

	// 首次创建配置文件，立即保存
	if fileExist == false {
		saveConfigFile(baseCfg)
	}
}

// initBaseConfig 初始化基础默认配置
func initBaseConfig(name, desc, version string, c IConfig) *Config {
	config := c.getBaseConfig()
	config.module = name
	config.desc = desc
	config.version = version
	config.filePath = "./config.yaml"
	config.Broker = struct {
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
	config.CallBack = struct {
		Notice string
		Log    string
	}{
		Notice: "All",
		Log:    "All",
	}

	return config
}

// setByArgs 根据外部传参更新基础配置
func setByArgs(config *Config) {
	defer func() {
		if r := recover(); r != nil {

		}
	}()

	// 如果有入参，则用入参（仅处理ConfigPath，其他参数在loadConfig中处理）
	if len(os.Args) > 1 {
		args := map[string]any{}
		err := json.Unmarshal([]byte(os.Args[1]), &args)
		if err == nil {
			// 自定义配置文件路径
			if val, ok := args["ConfigPath"]; ok {
				config.filePath = val.(string)
			}
			// 自定义模块名称
			if val, ok := args["Module"]; ok && val != "" {
				config.module = val.(string)
			}
			// 自定义Broker配置
			if val, ok := args["Broker"]; ok {
				err = json.Unmarshal([]byte(val.(string)), &config.Broker)
				if err != nil {
					panic(err)
				}
			}
		}
	}
	// 其他扩展参数
	if len(os.Args) > 2 {
		args := map[string]string{}
		for i := 2; i < len(os.Args); i += 2 {
			if i+1 >= len(os.Args) {
				break
			}
			key := os.Args[i]
			value := os.Args[i+1]
			args[key] = value
		}
		for key, value := range args {
			switch key {
			case "-exit":
				config.exit = value
			case "-port":
				// 重新设置端口
				port, _ := strconv.Atoi(value)
				a1, a2, a3, e := splitWebSocketURLRegex(config.Broker.Addr)
				if e == nil && strings.Contains(a2, ":") {
					sp := strings.Split(a2, ":")
					config.Broker.Addr = fmt.Sprintf("%s%s:%d%s", a1, sp[0], port, a3)
				}
			}
		}
	}
}

// splitWebSocketURLRegex 拆分broker连接地址
func splitWebSocketURLRegex(url string) (string, string, string, error) {
	pattern := `^(.+://)([^/]+)(/.+)$`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(url)
	if len(matches) != 4 {
		return "", "", "", fmt.Errorf("invalid URL format")
	}

	return matches[1], matches[2], matches[3], nil
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
