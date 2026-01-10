package test

import (
	"fmt"
	"github.com/kamioair/qf"
	"moduleA"
	"testing"
)

func TestName(t *testing.T) {
	// 需要先启动Broker再测试

	// 创建模块
	module1 := qf.NewModule(moduleA.NewService(map[string]any{"ModuleName": "ModuleA.Test1"}))
	module2 := qf.NewModule(moduleA.NewService(map[string]any{"ModuleName": "ModuleA.Test2"}))

	// 创建测试器
	test := qf.NewTest(module1, module2)

	// 测试业务功能
	respA, err := test.Invoke("ModuleA.Test1", "MethodA", nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respA.Raw())

	respB, err := test.Invoke("ModuleA.Test2", "MethodB", respA.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respB.Raw())

	respC, err := test.Invoke("ModuleA.Test2", "MethodC", respB.Raw())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(respC.Raw())

	// 不退出
	select {}
}
