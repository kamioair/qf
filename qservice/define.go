package qservice

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
	"github.com/kamioair/qf/utils/qconvert"
	"github.com/kamioair/qf/utils/qio"
	"os"
	"time"
)

const (
	routeModuleName = "Route"
)

// Setting 模块配置
type Setting struct {
	Module                  string                // 模块服务名称
	Desc                    string                // 模块服务描述
	Version                 string                // 模块服务版本
	DevCode                 string                // 设备码
	DevName                 string                // 设备名称
	Broker                  qdefine.BrokerConfig  // 主服务配置
	isAddModuleSuffix       bool                  // 模块是否附加设备id后缀
	onInitHandler           qdefine.InitHandler   // 初始化回调
	onReqHandler            qdefine.ReqHandler    // 请求回调
	onNoticeHandler         qdefine.NoticeHandler // 通知回调
	onStatusHandler         qdefine.NoticeHandler // 全局状态回调
	onCommStateHandler      qdefine.StateHandler  // 通讯状态回调
	onLoadServDiscoveryList func() string
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
	module := moduleName
	devCode := ""
	devName := ""
	mqAddr := ""
	// 根据传参更新配置
	if len(os.Args) > 1 {
		args := args{}
		err = json.Unmarshal([]byte(os.Args[1]), &args)
		if err != nil {
			panic(err)
		}
		mqAddr = args.MqAddr
		devCode = args.DeviceCode
		devName = args.DeviceName
		if args.Module != "" {
			module = args.Module
		}
		if args.ConfigPath != "" {
			configPath = args.ConfigPath
		}
	}
	// 设置配置文件路径
	qconfig.ChangeFilePath(configPath)
	broker := qdefine.BrokerConfig{
		Addr:    qconfig.Get(module, "mqtt.addr", "ws://127.0.0.1:5002/ws"),
		UId:     qconfig.Get(module, "mqtt.uid", ""),
		Pwd:     qconfig.Get(module, "mqtt.pwd", ""),
		LogMode: qconfig.Get(module, "mqtt.logMode", "NONE"),
		TimeOut: qconfig.Get(module, "mqtt.timeOut", 3000),
		Retry:   qconfig.Get(module, "mqtt.retry", 3),
	}
	if mqAddr != "" {
		broker.Addr = mqAddr
	}
	if devName == "" {
		if dev, err := DeviceCode.LoadFromFile(); err == nil {
			devName = dev.Name
		}
	}
	// 返回配置
	setting := &Setting{
		Module:  module,
		Desc:    moduleDesc,
		Version: version,
		Broker:  broker,
		DevCode: devCode,
		DevName: devName,
	}
	return setting
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

func (s *Setting) BindStatusFunc(onRetainNoticeHandler qdefine.NoticeHandler) *Setting {
	s.onStatusHandler = onRetainNoticeHandler
	return s
}

func (s *Setting) BindCommStateFunc(onStateHandler qdefine.StateHandler) *Setting {
	s.onCommStateHandler = onStateHandler
	return s
}

func (s *Setting) BindLoadServDiscoveryList(handler func() string) *Setting {
	s.onLoadServDiscoveryList = handler
	return s
}

func (s *Setting) SetDeviceCode(devCode string, isAddModuleSuffix bool) *Setting {
	s.DevCode = devCode
	s.isAddModuleSuffix = isAddModuleSuffix
	return s
}

type args struct {
	Module     string
	DeviceCode string
	DeviceName string
	ConfigPath string
	MqAddr     string
}

type servDiscovery struct {
	Id      string            // 服务器Broker所在设备ID
	Modules map[string]string // 包含的服务器模块和请求设备的模块列表，key为模块名称，value为设备Id，用于请求设备查找请求模块所在的设备
}

type runLog struct {
	Id      string // 来至设备ID
	Module  string // 来至模块ID
	Content string // 日志内容
}

func writeErrLog(tp string, err string) {
	logStr := fmt.Sprintf("DateTime: %s\n", qconvert.DateTime.ToString(time.Now(), "yyyy-MM-dd HH:mm:ss"))
	logStr += fmt.Sprintf("From: %s\n", tp)
	logStr += fmt.Sprintf("Error: %s\n", err)
	logStr += "----------------------------------------------------------------------------------------------\n\n"
	per := qconvert.DateTime.ToString(time.Now(), "yyyy-MM")
	day := qconvert.DateTime.ToString(time.Now(), "dd")
	logFile := fmt.Sprintf("./log/%s/%s_%s.log", per, day, "Error")
	logFile = qio.GetFullPath(logFile)
	_ = qio.WriteString(logFile, logStr, true)
}
