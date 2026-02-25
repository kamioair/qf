package qf

import (
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type TestService struct {
	Service
}

type ERunTestMode string

const (
	ERunTestModeCgoBroker  ERunTestMode = "CgoBroker"
	ERunTestModeMqttBroker ERunTestMode = "MqttBroker"
)

// RunTest 启动测试用例
func RunTest(mode ERunTestMode, services ...IService) *TestService {
	serv := &TestService{}
	serv.Load(fmt.Sprintf("QfTest.%d", time.Now().UnixNano()), "测试服务", "V1.0.260115B01", "", nil)

	switch mode {
	case ERunTestModeCgoBroker:
		broker := easyCon.NewCgoBroker()

		sm := newPluginTest(serv, broker.Publish)
		sm.RunAsync()
		broker.RegClient(sm.Name(), sm.onRead)

		for _, s := range services {
			tm := newPluginTest(s, broker.Publish)
			tm.RunAsync()

			broker.RegClient(tm.Name(), tm.onRead)
		}
	case ERunTestModeMqttBroker:
		sm := NewModule(serv)
		sm.RunAsync()

		for _, s := range services {
			tm := NewModule(s)
			tm.RunAsync()
		}
	default:
		panic("不支持的测试模式")
	}

	return serv
}
