package moduleA

import (
	"fmt"
	"github.com/kamioair/qf"
)

type bll struct {
}

func newBll() *bll {
	return &bll{}
}

func (b *bll) MethodA(ctx qf.IContext) (any, error) {
	return "hello methodA", nil
}

func (b *bll) MethodB(ctx qf.IContext) (any, error) {
	req := ctx.Raw()
	resp := TestInfo{
		Name: "MethodB",
		Info: fmt.Sprintf("from req %s", req),
	}
	return resp, nil
}

func (b *bll) MethodC(ctx qf.IContext) (any, error) {
	req := TestInfo{}
	if err := ctx.Bind(&req); err != nil {
		return nil, err
	}

	return fmt.Sprintf("methodC req=%s,%s", req.Name, req.Info), nil
}
