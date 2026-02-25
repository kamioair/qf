package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// BindRequest 绑定请求方法
func BindRequest[Req any, Resp any](service IService, route string, method func(Req) (Resp, easyCon.EResp, error)) {
	// 实现逻辑...
	handler := &genericHandler[Req, Resp]{service: service, route: route, requestFunc: method}
	service.bindRequest(route, handler)
}

// BindNotice 绑定通知方法
func BindNotice[Req any](service IService, route string, method func(Req), isRetain bool) {
	handler := &genericHandler[Req, Void]{service: service, route: route, noticeFunc: method}
	service.bindNotice(route, handler, isRetain)
}

// SendRequest 向其他模块发送请求
func SendRequest[Req any, Resp any](service IService, module string, route string, param Req) (Resp, easyCon.EResp, error) {
	var respContent Resp

	// 转换入参
	reqContent, err := anyToBytes[Req](param)
	if err != nil {
		return respContent, easyCon.ERespError, err
	}

	// 执行请求
	resp := service.adapter().Req(module, route, reqContent)

	// 返回请求结果
	if resp.RespCode != easyCon.ERespSuccess {
		return respContent, resp.RespCode, errors.New(string(resp.Content))
	}
	respContent, err = bytesToAny[Resp](resp.Content)
	if err != nil {
		return respContent, easyCon.ERespError, err
	}
	return respContent, easyCon.ERespSuccess, nil
}

// SendRequestWithTimeout 向其他模块发送请求(指定超时时间)
func SendRequestWithTimeout[Req any, Resp any](service IService, module string, route string, param Req, timeout int) (Resp, easyCon.EResp, error) {
	var respContent Resp

	// 转换入参
	reqContent, err := anyToBytes[Req](param)
	if err != nil {
		return respContent, easyCon.ERespError, err
	}

	// 执行请求
	resp := service.adapter().ReqWithTimeout(module, route, reqContent, timeout)

	// 返回请求结果
	if resp.RespCode != easyCon.ERespSuccess {
		return respContent, resp.RespCode, errors.New(string(resp.Content))
	}
	respContent, err = bytesToAny[Resp](resp.Content)
	if err != nil {
		return respContent, easyCon.ERespError, err
	}
	return respContent, easyCon.ERespSuccess, nil
}

// SendNotice 发送通知
func SendNotice[Req any](service IService, route string, param Req, isRetain bool) error {
	// 转换入参
	reqContent, err := anyToBytes[Req](param)
	if err != nil {
		return err
	}

	// 执行请求
	if isRetain == false {
		err = service.adapter().SendNotice(route, reqContent)
		if err != nil {
			return err
		}
	} else {
		err = service.adapter().SendRetainNotice(route, reqContent)
		if err != nil {
			return err
		}
	}

	return nil
}

// 具体的请求处理器
type genericHandler[Req any, Resp any] struct {
	service     IService
	route       string
	requestFunc func(Req) (Resp, easyCon.EResp, error)
	noticeFunc  func(Req)
}

// Handle 执行方法处理
func (h *genericHandler[Req, Resp]) Handle(data []byte) ([]byte, easyCon.EResp, error) {
	// 解析入参
	req, err := bytesToAny[Req](data)
	if err != nil {
		return nil, easyCon.ERespError, err
	}
	// 调用请求方法
	if h.requestFunc != nil {
		resp, code, err := h.requestFunc(req)
		if err != nil || code != easyCon.ERespSuccess {
			if code == easyCon.ERespError {
				str, _ := json.Marshal(req)
				writeLog(h.service.Name(), "Error", fmt.Sprintf("[%s] InParam=%s", h.route, str), err.Error())
			}
			return nil, code, err
		}
		// 解析返回值
		content, err := anyToBytes[Resp](resp)
		if err != nil {
			str, _ := json.Marshal(req)
			writeLog(h.service.Name(), "Error", fmt.Sprintf("[%s] InParam=%s", h.route, str), err.Error())
			return nil, easyCon.ERespError, err
		}
		return content, easyCon.ERespSuccess, nil
	}

	// 调用通知
	h.noticeFunc(req)
	return nil, easyCon.ERespSuccess, nil
}

func anyToBytes[T any](obj T) ([]byte, error) {
	if any(obj) == nil {
		return nil, nil
	}

	if s, ok := any(obj).(string); ok {
		// 如果是字符串，直接转换为 []byte
		return []byte(s), nil
	} else if b, ok := any(obj).([]byte); ok {
		// 如果已经是 []byte，直接使用
		return b, nil
	} else if _, ok := any(obj).(Void); ok {
		// 如果已经是 []byte，直接使用
		return nil, nil
	}
	// 其他类型（结构体等）转换为 JSON 格式的 []byte
	resp, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func bytesToAny[T any](data []byte) (T, error) {
	var result T

	// 检查 T 的类型
	var zero T

	// 如果是 string 类型
	if _, ok := any(zero).(string); ok {
		// 直接将 []byte 转换为 string
		strResult := string(data)
		return any(strResult).(T), nil
	}

	// 如果是 []byte 类型
	if _, ok := any(zero).([]byte); ok {
		return any(data).(T), nil
	}

	if _, ok := any(zero).(Void); ok {
		return zero, nil
	}

	// 其他类型使用 json.Unmarshal
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, err
	}

	return result, nil
}
