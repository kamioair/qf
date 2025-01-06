package qservice

import (
	"encoding/json"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
	"github.com/kamioair/qf/utils/qio"
	"os"
)

const (
	routeModuleName = "Route"
)

type EServerMode string

func (mode EServerMode) IsClient() bool {
	return mode == EModeClient
}

func (mode EServerMode) IsServer() bool {
	return mode == EModeServer
}

const (
	EModeClient EServerMode = "client"
	EModeServer EServerMode = "server"
)

// Setting 模块配置
type Setting struct {
	Mode                    EServerMode             // 路由模式
	Module                  string                  // 模块服务名称
	Desc                    string                  // 模块服务描述
	Version                 string                  // 模块服务版本
	DevCode                 string                  // 设备码
	Broker                  qdefine.BrokerConfig    // 主服务配置
	DetectedRoutes          []string                // 需要对外暴露的方法列表
	WatchedModules          []string                // 需要监听暴露的模块列表
	onInitHandler           qdefine.InitHandler     // 初始化回调
	onReqHandler            qdefine.ReqHandler      // 请求回调
	onNoticeHandler         qdefine.NoticeHandler   // 通知回调
	onStatusHandler         qdefine.NoticeHandler   // 全局状态回调
	onAcceptDetectedHandler qdefine.DetectedHandler // 接收暴露的路由执行内容
	onCommStateHandler      qdefine.StateHandler    // 通讯状态回调
}

type LogConfig struct {
	Path        string
	RemainDay   int
	RemainLevel int
}

// NewSetting 创建模块配置
func NewSetting(moduleName, moduleDesc, version string) *Setting {
	// 修改系统路径为当前目录
	err := os.Chdir(qio.GetCurrentDirectory())
	if err != nil {
		panic(err)
	}

	// 默认值
	configPath := "./config/config.yaml"
	errorLogPath = "./log"
	module := moduleName
	devCode := ""
	brokerArg := qdefine.BrokerConfig{}
	// 根据传参更新配置
	if len(os.Args) > 1 {
		args := Args{}
		err = json.Unmarshal([]byte(os.Args[1]), &args)
		if err != nil {
			panic(err)
		}
		brokerArg = args.Broker
		devCode = args.DeviceCode
		if args.Module != "" {
			module = args.Module
		}
		if args.ConfigPath != "" {
			configPath = args.ConfigPath
		}
		if args.LogPath != "" {
			errorLogPath = args.LogPath
		}
	}
	// 设置配置文件路径
	qconfig.ChangeFilePath(configPath)
	broker := qdefine.BrokerConfig{
		Addr:    qconfig.Get("", "mqtt.addr", "ws://127.0.0.1:5002/ws"),
		UId:     qconfig.Get("", "mqtt.uid", ""),
		Pwd:     qconfig.Get("", "mqtt.pwd", ""),
		LogMode: qconfig.Get("", "mqtt.logMode", "NONE"),
		TimeOut: qconfig.Get("", "mqtt.timeOut", 3000),
		Retry:   qconfig.Get("", "mqtt.retry", 3),
	}
	if brokerArg.Addr != "" {
		broker = brokerArg
	}
	// 返回配置
	setting := &Setting{
		Mode:           EServerMode(qconfig.Get("", "mode", "client")),
		DetectedRoutes: qconfig.Get(module, "detectedRoutes", []string{}),
		WatchedModules: qconfig.Get(module, "watchedModules", []string{}),
		Module:         module,
		Desc:           moduleDesc,
		Version:        version,
		Broker:         broker,
		DevCode:        devCode,
	}
	return setting
}

func (s *Setting) ReloadByCustomArgs(args Args) {
	if args.Module != "" {
		s.Module = args.Module
	}
	if args.ConfigPath != "" {
		qconfig.ChangeFilePath(args.ConfigPath)
		s.Broker = qdefine.BrokerConfig{
			Addr:    qconfig.Get("", "mqtt.addr", "ws://127.0.0.1:5002/ws"),
			UId:     qconfig.Get("", "mqtt.uid", ""),
			Pwd:     qconfig.Get("", "mqtt.pwd", ""),
			LogMode: qconfig.Get("", "mqtt.logMode", "NONE"),
			TimeOut: qconfig.Get("", "mqtt.timeOut", 3000),
			Retry:   qconfig.Get("", "mqtt.retry", 3),
		}
	}
	if args.Broker.Addr != "" {
		s.Broker = args.Broker
	}
	if args.DeviceCode != "" {
		s.DevCode = args.DeviceCode
	}
	if args.LogPath != "" {
		errorLogPath = args.LogPath
	}
}

func (s *Setting) BindInitFunc(onInitHandler qdefine.InitHandler) *Setting {
	s.onInitHandler = onInitHandler
	return s
}

func (s *Setting) BindReqFunc(onReqHandler qdefine.ReqHandler) *Setting {
	s.onReqHandler = onReqHandler
	return s
}

func (s *Setting) BindNoticeFunc(onNoticeHandler qdefine.NoticeHandler) *Setting {
	s.onNoticeHandler = onNoticeHandler
	return s
}

func (s *Setting) BindRespDetectedFunc(onDetectedHandler qdefine.DetectedHandler) *Setting {
	s.onAcceptDetectedHandler = onDetectedHandler
	return s
}

//func (s *Setting) BindStatusFunc(onRetainNoticeHandler qdefine.NoticeHandler) *Setting {
//	s.onStatusHandler = onRetainNoticeHandler
//	return s
//}

func (s *Setting) BindCommStateFunc(onStateHandler qdefine.StateHandler) *Setting {
	s.onCommStateHandler = onStateHandler
	return s
}

type Args struct {
	Module     string
	Broker     qdefine.BrokerConfig
	DeviceCode string
	DeviceName string
	ConfigPath string
	LogPath    string
}

//type servDiscovery struct {
//	Id      string            // 服务器Broker所在设备ID
//	Modules map[string]string // 包含的服务器模块和请求设备的模块列表，key为模块名称，value为设备Id，用于请求设备查找请求模块所在的设备
//}

type runLog struct {
	Id      string // 来至设备ID
	Module  string // 来至模块ID
	Content string // 日志内容
}
