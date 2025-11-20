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

type IConfig interface {
	getBaseConfig() *Config
}

type Config struct {
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

var (
	baseCfg *Config
)

// GetModuleInfo 获取基础配置（给外部用）
func (c *Config) GetModuleInfo() (Name string, Desc string, Version string) {
	return c.module, c.desc, c.version
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
	baseCfg = initBaseConfig(name, desc, version, config)
	fileExist := qio.PathExists(baseCfg.filePath)
	err = qconfig.LoadConfig(baseCfg.filePath, "Base", baseCfg)
	if err != nil {
		panic(err)
	}
	// 如果有外部传入参数，则更新配置
	setByArgs(baseCfg)

	// 加载模块自定义配置
	err = qconfig.LoadConfig(baseCfg.filePath, name, config)
	if err != nil {
		panic(err)
	}

	// 首次创建配置文件，立即保存
	if fileExist == false {
		saveConfigFile()
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

	return config
}

// setByArgs 根据外部传参更新基础配置
func setByArgs(config *Config) {
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
func saveConfigFile() {
	if baseCfg == nil {
		return
	}

	// 准备保存选项
	opts := qconfig.SaveConfigOptions{
		SectionDescs: map[string]string{
			baseCfg.module: baseCfg.desc,
		},
	}

	// 保存配置
	err := qconfig.SaveConfig(baseCfg.filePath, &opts)
	if err != nil {
		fmt.Printf("保存配置文件失败: %v\n", err)
	}
}
