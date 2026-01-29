package test

import (
	"fmt"
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"testing"
)

func TestName(t *testing.T) {
	// 创建本项目服务
	exampleServ := example.NewService()

	// 启动测试用例
	testServ := qf.RunTest(qf.ERunTestModeCgoBroker, exampleServ)

	// 测试业务功能
	respA := testServ.SendRequest(exampleServ.Name(), "MethodA", []byte("hello methodA"))
	if respA.RespCode != easyCon.ERespSuccess {
		t.Fatal(respA.Content)
	}
	fmt.Println("===> SendRequest MethodA Resp", respA.Content)

	respB := testServ.SendRequest(exampleServ.Name(), "MethodB", respA.Content)
	if respB.RespCode != easyCon.ERespSuccess {
		t.Fatal(respB.Content)
	}
	fmt.Println("===> SendRequest MethodB Resp", respB.Content)

	respC := testServ.SendRequest(exampleServ.Name(), "MethodC", respB.Content)
	if respC.RespCode != easyCon.ERespSuccess {
		t.Fatal(respC.Content)
	}
	fmt.Println("===> SendRequest MethodC Resp", respC.Content)

	// 不退出
	select {}
}
