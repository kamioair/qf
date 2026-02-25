package test

import (
	"fmt"
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	"testing"
)

func TestName(t *testing.T) {
	// 创建本项目服务
	exampleServ := example.NewService()

	// 启动测试用例
	testServ := qf.RunTest(qf.ERunTestModeCgoBroker, exampleServ)

	// 测试业务功能
	respA, codeA, errA := qf.SendRequest[string, *example.TestInfo](testServ, exampleServ.Name(), "MethodA", "test")
	fmt.Println("===> SendRequest MethodA Resp", respA, codeA, errA)

	respB, codeB, errB := qf.SendRequest[qf.Void, []string](testServ, exampleServ.Name(), "MethodB", nil)
	fmt.Println("===> SendRequest MethodB Resp", respB, codeB, errB)

	respC, codeC, errC := qf.SendRequest[string, qf.Void](testServ, exampleServ.Name(), "MethodC", "test")
	fmt.Println("===> SendRequest MethodC Resp", respC, codeC, errC)

	errN := qf.SendNotice[string](testServ, "ChangedNotice", "10001", false)
	fmt.Println("===> SendNotice ChangedNotice Resp", errN)

	// 不退出
	select {}
}
