package moduleA

import "github.com/kamioair/qf"

type bll struct {
}

func newBll() *bll {
	return &bll{}
}

func (b *bll) MethodA(ctx qf.IContext) (any, error) {
	return "hello methodA", nil
}
