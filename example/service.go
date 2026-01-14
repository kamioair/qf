package example

import (
	"github.com/kamioair/qf"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

const (
	Version = "V1.0.251225B01"
	Name    = "ModuleA"
	Desc    = "模板模块A"
)

// Service 模块服务入口
type Service struct {
	qf.Service
	cfg *Config

	// 具体业务功能实现
	bll *bll
}

// Config 自定义配置
type Config struct {
	qf.Config

	// 自定义配置
	// ...
}

// NewService 创建功能实现入口
func NewService(customSetting map[string]any) *Service {
	serv := &Service{
		cfg: &Config{
			// 自定义配置初始值
			// ...
		},
	}
	// 加载配置
	serv.Load(Name, Desc, Version, serv.cfg, customSetting)
	return serv
}

// Reg 注册需要执行的方法
func (serv *Service) Reg(reg *qf.Reg) {
	reg.OnInit = serv.onInit
	reg.OnReq = serv.onReq
}

// 初始化
func (serv *Service) onInit() {
	// 内部业务初始化
	serv.bll = newBll()
}

// 实现外部请求
func (serv *Service) onReq(pack easyCon.PackReq) (easyCon.EResp, any) {
	switch pack.Route {
	case "MethodA":
		return qf.Invoke(pack, serv.bll.MethodA)
	case "MethodB":
		return qf.Invoke(pack, serv.bll.MethodB)
	case "MethodC":
		return qf.Invoke(pack, serv.bll.MethodC)
	}
	return serv.ReturnNotFind()
}
