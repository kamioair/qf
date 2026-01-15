package test

import (
	"fmt"
	"github.com/kamioair/qf"
	"github.com/kamioair/qf/example"
	"testing"
)

func TestName(t *testing.T) {
	// 需要先启动Broker再测试

	// 创建模块
	module := qf.NewModule(example.NewService())

	// 创建测试器
	test := qf.NewTest(module)

	// 测试业务功能
	respA, err := test.Invoke(module.Name(), "MethodA", "hello methodA")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respA.Raw())

	respB, err := test.Invoke(module.Name(), "MethodB", respA.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respB.Raw())

	respC, err := test.Invoke(module.Name(), "MethodC", respB.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respC.Raw())

	// 不退出
	select {}
}
