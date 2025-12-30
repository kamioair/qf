package moduleA

import (
	"github.com/kamioair/qf"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// Service 模块服务入口
type Service struct {
	qf.Service
	cfg *ConfigStruct

	// 具体业务功能实现
	bll *bll
}

func NewService(cfg *ConfigStruct) *Service {
	serv := &Service{cfg: cfg}
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
		return serv.Invoke(pack, serv.bll.MethodA)
	}
	return serv.ReturnNotFind()
}
