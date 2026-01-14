package qf

type Test struct {
	testModule  IModule
	testService *testService
	modules     []IModule
}

type testService struct {
	Service
}

func (t *testService) Reg(reg *Reg) {

}

// NewTest 创建测试用例, 入参为需要测试的模块
func NewTest(modules ...IModule) *Test {
	t := &Test{
		modules:     modules,
		testService: &testService{},
	}
	t.testService.Load("TestModule", "测试模块", "1.0.0", nil, nil)

	// 创建用例模块
	t.testModule = NewModule(t.testService)
	t.testModule.RunAsync()

	// 启动测试模块
	for _, m := range modules {
		m.RunAsync()
	}

	return t
}

// Invoke 执行方法
func (t *Test) Invoke(moduleName string, route string, params any) (IContext, error) {
	return t.testService.SendRequestWithTimeout(moduleName, route, params, 60000)
}

// InvokeWithTimeout 执行方法，并自定义超时时间，单位毫秒
func (t *Test) InvokeWithTimeout(moduleName string, route string, params any, timeout int) (IContext, error) {
	return t.testService.SendRequestWithTimeout(moduleName, route, params, timeout)
}
