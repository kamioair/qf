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
	// 安全检查：确保 refStruct 不为 nil
	if refStruct == nil {
		return fmt.Errorf("refStruct cannot be nil")
	}

	raw := c.raw
	// 先尝试直接解析为JSON
	err := json.Unmarshal([]byte(raw), refStruct)
	if err == nil {
		return nil
	}

	// 如果JSON解析失败，尝试智能转换
	switch v := refStruct.(type) {
	case *string:
		// 额外检查：确保指针本身不为 nil
		if v == nil {
			return fmt.Errorf("string pointer is nil")
		}
		*v = raw
		return nil
	case *int:
		if v == nil {
			return fmt.Errorf("int pointer is nil")
		}
		num, err := strconv.Atoi(raw)
		if err != nil {
			return err
		}
		*v = num
		return nil
	case *float64:
		if v == nil {
			return fmt.Errorf("float64 pointer is nil")
		}
		num, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		*v = num
		return nil
	case *bool:
		if v == nil {
			return fmt.Errorf("bool pointer is nil")
		}
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
