package qf

import (
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"os"
	"time"
)

type TestService struct {
	Service
}

func (t *Service) Reg(reg *Reg) {
	reg.OnInit = t.onInit
	reg.OnReq = t.onReq
}

func (t *Service) onInit() {

}

func (t *Service) onReq(pack easyCon.PackReq) (easyCon.EResp, any) {
	return t.ReturnNotFind()
}

func NewTest(moduleName string, moduleService IService, moduleConfig IConfig) *TestService {
	os.Args = []string{os.Args[0]}
	// 启动待调试的模块
	run := NewModule(moduleName, moduleName, "TestVersion", moduleService, moduleConfig)
	go run.Run()

	service := &TestService{}
	module := NewModule(fmt.Sprintf("QfTest.%d", time.Now().UnixNano()), "测试服务", "V1.0.260115B01", service, &config)
	go module.Run()

	// 等待片刻，确保服务启动完成
	time.Sleep(time.Second * 2)

	return service
}

// config 配置定义
var config = struct {
	Config

	// 自己模块的配置
}{}
