package example

import (
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

type bll struct {
}

func newBll() *bll {
	return &bll{}
}

func (b *bll) MethodA(req string) (string, easyCon.EResp, error) {
	resp := "hello methodA"
	return resp, easyCon.ERespSuccess, nil
}

func (b *bll) MethodB(req string) (TestInfo, easyCon.EResp, error) {
	resp := TestInfo{
		Name: "MethodB",
		Info: fmt.Sprintf("from req %s", req),
	}
	return resp, easyCon.ERespSuccess, nil
}

func (b *bll) MethodC(req TestInfo) (TestInfo, easyCon.EResp, error) {
	resp := TestInfo{
		Name: "MethodC",
		Info: fmt.Sprintf("from req %s", req),
	}
	return resp, easyCon.ERespSuccess, nil
}
