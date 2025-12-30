package moduleA

import "github.com/kamioair/qf"

const (
	Version = "V1.0.251225B01"
	Name    = "ModuleA"
	Desc    = "模板模块A"
)

type ConfigStruct struct {
	qf.Config

	// 自定义配置
	// ...
}

func NewConfig() *ConfigStruct {
	return &ConfigStruct{
		// 自定义配置初始值
		// ...
	}
}
