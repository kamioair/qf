package qf

import (
	"encoding/json"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// IContext 上下文
type IContext interface {
	Raw() string
	Bind(refStruct any)
}

type context struct {
	raw        string
	reqPack    *easyCon.PackReq
	respPack   *easyCon.PackResp
	noticePack *easyCon.PackNotice
}

// NewContent 创建上下文
func NewContent(value any) (IContext, error) {
	return newContent(value, nil, nil, nil)
}

func newContent(value any, reqPack *easyCon.PackReq, respPack *easyCon.PackResp, noticePack *easyCon.PackNotice) (IContext, error) {
	var raw string

	switch v := value.(type) {
	case string:
		raw = v
	default:
		js, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		raw = string(js)
	}

	ctx := &context{
		raw:        raw,
		reqPack:    reqPack,
		respPack:   respPack,
		noticePack: noticePack,
	}
	return ctx, nil
}

func (c *context) Raw() string {
	return c.raw
}

func (c *context) Bind(refStruct any) {
	err := json.Unmarshal([]byte(c.raw), refStruct)
	if err != nil {
		panic(err)
	}
}

func formatRespError(respCode easyCon.EResp, errStr string) string {
	respDesc := fmt.Sprintf("%d", respCode)
	switch respCode {
	case easyCon.ERespUnLinked:
		respDesc = "UnLinked"
	case easyCon.ERespSuccess:
		respDesc = "Success"
	case easyCon.ERespBadReq:
		respDesc = "BadReq"
	case easyCon.ERespRouteNotFind:
		respDesc = "RouteNotFind"
	case easyCon.ERespError:
		respDesc = "Error"
	case easyCon.ERespTimeout:
		respDesc = "Timeout"
	}
	return fmt.Sprintf("RespCode=%d(%s), Error=%s", respCode, respDesc, errStr)
}
