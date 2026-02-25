package example

import (
	"errors"
	"fmt"
	"github.com/kamioair/qf"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

type bll struct {
}

func (b bll) MethodA(key string) (*TestInfo, easyCon.EResp, error) {
	if key != "test" {
		return nil, easyCon.ERespError, errors.New("key value != test")
	}
	return &TestInfo{
		Name: "1",
		Info: "2",
	}, easyCon.ERespSuccess, nil
}

func (b bll) MethodB(void qf.Void) ([]string, easyCon.EResp, error) {
	return []string{"1", "2", "3"}, easyCon.ERespSuccess, nil
}

func (b bll) MethodC(key string) (qf.Void, easyCon.EResp, error) {
	return nil, easyCon.ERespSuccess, nil
}

func (b bll) ChangedNotice(id string) {
	fmt.Println(id)
}

func newBll() *bll {
	return &bll{}
}
