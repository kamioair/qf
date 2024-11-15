package qservice

import (
	"encoding/json"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
	"github.com/kamioair/qf/utils/qio"
	"os"
	"strings"
)

const (
	clientManageName = "ClientManager"
)

// Setting 模块配置
type Setting struct {
	Module          string                // 模块服务名称
	Desc            string                // 模块服务描述
	Version         string                // 模块服务版本
	DevCode         string                // 设备码
	Broker          qdefine.BrokerConfig  // 主服务配置
	onInitHandler   qdefine.InitHandler   // 初始化回调
	onReqHandler    qdefine.ReqHandler    // 请求回调
	onNoticeHandler qdefine.NoticeHandler // 通知回调
	onStateHandler  qdefine.StateHandler  // 状态回调
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
		UId:     qconfig.Get(module, "mqtt.username", ""),
		Pwd:     qconfig.Get(module, "mqtt.password", ""),
		LogMode: qconfig.Get(module, "mqtt.logMode", "NONE"),
		TimeOut: qconfig.Get(module, "mqtt.timeOut", 3000),
		Retry:   qconfig.Get(module, "mqtt.retry", 3),
	}
	if mqAddr != "" {
		broker.Addr = mqAddr
	}
	// 返回配置
	return &Setting{
		Module:  module,
		Desc:    moduleDesc,
		Version: version,
		Broker:  broker,
		DevCode: devCode,
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

func (s *Setting) BindStateFunc(onStateHandler qdefine.StateHandler) *Setting {
	s.onStateHandler = onStateHandler
	return s
}

func (s *Setting) SetDeviceCode(code string) *Setting {
	s.DevCode = code
	return s
}

type args struct {
	Module     string
	DeviceCode string
	ConfigPath string
	MqAddr     string
}

func newModuleName(module, code string) string {
	sp := strings.Split(module, ".")
	if len(sp) >= 2 {
		module = sp[0] + "." + code
	} else {
		module = module + "." + code
	}
	module = strings.Trim(module, ".")
	return module
}

func writeErrLog(tp string, err string) {

}
