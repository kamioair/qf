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
	respA, err := testServ.SendRequest(exampleServ.Name(), "MethodA", "hello methodA")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("===> SendRequest MethodA Resp", respA.Raw())

	respB, err := testServ.SendRequest(exampleServ.Name(), "MethodB", respA.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("===> SendRequest MethodB Resp", respB.Raw())

	respC, err := testServ.SendRequest(exampleServ.Name(), "MethodC", respB.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("===> SendRequest MethodC Resp", respC.Raw())

	// 不退出
	select {}
}
