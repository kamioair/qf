package qf

import (
	"encoding/json"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"strconv"
)

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

func (c *context) Bind(refStruct any) error {
	raw := c.raw

	// 先尝试直接解析为JSON
	err := json.Unmarshal([]byte(raw), refStruct)
	if err == nil {
		return nil
	}

	// 如果JSON解析失败，尝试智能转换
	switch v := refStruct.(type) {
	case *string:
		*v = raw
		return nil
	case *int:
		num, err := strconv.Atoi(raw)
		if err != nil {
			return err
		}
		*v = num
		return nil
	case *float64:
		num, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		*v = num
		return nil
	case *bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		*v = b
		return nil
	default:
		// 对于其他类型，返回原始的JSON错误
		return err
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
