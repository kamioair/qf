package qf

import (
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"time"
)

type TestService struct {
	Service
}

func (serv *TestService) Reg(reg *Reg) {
	reg.OnInit = serv.onInit
	reg.OnReq = serv.onReq
}

func (serv *TestService) onInit() {

}

func (serv *TestService) onReq(pack easyCon.PackReq) (easyCon.EResp, []byte) {
	return serv.ReturnNotFind()
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

//type Test struct {
//	testModule  IModule
//	testService *testService
//	modules     []IModule
//}
//
//type testService struct {
//	Service
//}
//
//func (t *testService) Reg(reg *Reg) {
//
//}
//
//// NewTest 创建测试用例, 入参为需要测试的模块
//func NewTest(modules ...IModule) *Test {
//	t := &Test{
//		modules:     modules,
//		testService: &testService{},
//	}
//	t.testService.Load("TestModule", "测试模块", "1.0.0", "TestModule", nil)
//
//	// 创建用例模块
//	t.testModule = NewModule(t.testService)
//	t.testModule.RunAsync()
//
//	// 启动测试模块
//	for _, m := range modules {
//		m.RunAsync()
//	}
//
//	return t
//}
//
//// Invoke 执行方法
//func (t *Test) Invoke(moduleName string, route string, params any) (IContext, error) {
//	return t.testService.SendRequestWithTimeout(moduleName, route, params, 60000)
//}
//
//// InvokeWithTimeout 执行方法，并自定义超时时间，单位毫秒
//func (t *Test) InvokeWithTimeout(moduleName string, route string, params any, timeout int) (IContext, error) {
//	return t.testService.SendRequestWithTimeout(moduleName, route, params, timeout)
//}
