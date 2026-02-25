package example

import (
	"github.com/kamioair/qf"
)

const (
	Version = "V1.0.251225B01"
	Name    = "ExampleModule"
	Desc    = "模板模块"
)

// Service 模块服务入口
type Service struct {
	qf.Service
	cfg *Config

	// 其他自定义业务
	bll *bll
}

// Config 自定义配置
type Config struct {
	qf.Config

	// 自定义配置
	// ...
}

// NewService 创建功能实现入口
func NewService() *Service {
	serv := &Service{
		cfg: &Config{
			// 自定义配置初始值
			// ...
		},
	}
	// 加载配置，默认配置节点名称为模块名
	// 如有需求可自定义customSectionName
	serv.Load(Name, Desc, Version, "", serv.cfg)

	// 初始化业务逻辑
	serv.bll = newBll()

	// 注册路由
	qf.BindRequest[string, *TestInfo](serv, "MethodA", serv.bll.MethodA) // 有入参，有返回
	qf.BindRequest[qf.Void, []string](serv, "MethodB", serv.bll.MethodB) // 无入参，有返回
	qf.BindRequest[string, qf.Void](serv, "MethodC", serv.bll.MethodC)   // 有入参，无返回

	// 注册通知
	qf.BindNotice[string](serv, "ChangedNotice", serv.bll.ChangedNotice, false)

	return serv
}

//
//// Reg 注册需要执行的方法
//func (serv *Service) Reg(reg *qf.Reg) {
//	reg.OnInit = serv.onInit
//	reg.OnReq = serv.onReq
//	reg.RegNotice("22",serv.onNotice, )
//
//	qf.Reg(serve, “”)
//}
//
//// 初始化
//func (serv *Service) onInit() {
//	// 内部业务初始化
//	serv.bll = newBll()
//}
//
//// 实现外部请求
//func (serv *Service) onReq(pack easyCon.PackReq) (easyCon.EResp, []byte) {
//	switch pack.Route {
//	case "MethodA":
//		return qf.Invoke(pack, serv.bll.MethodA)
//	case "MethodB":
//		return qf.Invoke(pack, serv.bll.MethodB)
//	case "MethodC":
//		return qf.Invoke(pack, serv.bll.MethodC)
//	}
//	return serv.ReturnNotFind()
//}

//func (serv *Service) onNotice(notice easyCon.PackNotice) {
//	notice.Route
//
//}
