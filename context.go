package qf

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gobeam/stringy"
	"github.com/kamioair/utils/qtime"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// File 文件
type File struct {
	Name string // 文件名
	Size int64  // 文件大小
	Data []byte // 内容
}

// Context 上下文
// 取消原来的 IContext 接口，直接以具体类型对外提供能力；
// 旧 CommPack 被 easyCon.PackReq / easyCon.PackNotice 取代，由 GetPackReq / GetPackNotice 暴露。
type Context struct {
	values *values
	req    *easyCon.PackReq
	notice *easyCon.PackNotice
}

type values struct {
	InputMaps   []map[string]interface{}
	InputRaw    interface{}
	OutputValue interface{}
}

// NewContext 创建上下文
// value 为原始入参（任意可序列化类型）；reqPack / noticePack 至少传一个，由调用场景决定。
func NewContext(value any, reqPack *easyCon.PackReq, noticePack *easyCon.PackNotice) (*Context, error) {
	ctx := &Context{
		values: &values{
			InputMaps: make([]map[string]interface{}, 0),
		},
	}
	if reqPack != nil {
		ctx.req = reqPack
	}
	if noticePack != nil {
		ctx.notice = noticePack
	}
	if err := setData(ctx, value); err != nil {
		return nil, err
	}
	return ctx, nil
}

func setData(ctx *Context, data any) error {
	if data != nil {
		var content []byte
		switch data.(type) {
		case string:
			str := data.(string)
			if (strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")) ||
				strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]") {
				content = []byte(str)
			} else {
				content = []byte(fmt.Sprintf("\"%s\"", str))
			}
		default:
			js, err := json.Marshal(data)
			if err != nil {
				return err
			}
			content = js
		}
		err := ctx.values.load(content)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get 泛型取值
//   - key 为空：将整个入参转为 T（结构体取单条；[]X 取批量；反序列化失败 panic）
//   - key 非空：从入参取指定 key 的值并转为 T（找不到或转换失败返回 T 的零值）
//
// 须显式指定类型实参：
//
//	ctx.Get[string]("name")
//	ctx.Get[User]("")
//	ctx.Get[[]User]("")
func (c *Context) Get[T any](key string) T {
	var zero T

	if key == "" {
		// 整体反序列化：按 T 的 Kind 选单条或批量
		t := reflect.TypeOf(zero)
		var raw any
		if t != nil && t.Kind() == reflect.Slice {
			raw = c.values.InputMaps
		} else {
			raw = c.values.InputRaw
		}
		js, err := json.Marshal(raw)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(js, &zero); err != nil {
			panic(err)
		}
		return zero
	}

	// 单字段取值
	raw := c.values.getValue(key)
	if raw == nil {
		return zero
	}
	return convertTo[T](raw)
}

// GetPackReq 返回请求包副本；无请求包时返回零值
func (c *Context) GetPackReq() easyCon.PackReq {
	if c.req != nil {
		return *c.req
	}
	return easyCon.PackReq{}
}

// GetPackNotice 返回通知包副本；无通知包时返回零值
func (c *Context) GetPackNotice() easyCon.PackNotice {
	if c.notice != nil {
		return *c.notice
	}
	return easyCon.PackNotice{}
}

// Raw 返回原始入参（与旧版本保持一致）
func (c *Context) Raw() any {
	return c.values.InputRaw
}

// convertTo 按 T 的具体类型把 any 转换为 T
func convertTo[T any](v any) T {
	var zero T
	switch any(zero).(type) {
	case string:
		return any(toString(v)).(T)
	case int:
		n, _ := strconv.Atoi(toString(v))
		return any(n).(T)
	case int8:
		n, _ := strconv.ParseInt(toString(v), 10, 8)
		return any(int8(n)).(T)
	case int16:
		n, _ := strconv.ParseInt(toString(v), 10, 16)
		return any(int16(n)).(T)
	case int32:
		n, _ := strconv.ParseInt(toString(v), 10, 32)
		return any(int32(n)).(T)
	case int64:
		n, _ := strconv.ParseInt(toString(v), 10, 64)
		return any(n).(T)
	case uint:
		n, _ := strconv.ParseUint(toString(v), 10, 64)
		return any(uint(n)).(T)
	case uint8: // == byte
		n, _ := strconv.ParseUint(toString(v), 10, 8)
		return any(uint8(n)).(T)
	case uint16:
		n, _ := strconv.ParseUint(toString(v), 10, 16)
		return any(uint16(n)).(T)
	case uint32:
		n, _ := strconv.ParseUint(toString(v), 10, 32)
		return any(uint32(n)).(T)
	case uint64:
		n, _ := strconv.ParseUint(toString(v), 10, 64)
		return any(n).(T)
	case float32:
		n, _ := strconv.ParseFloat(toString(v), 32)
		return any(float32(n)).(T)
	case float64:
		n, _ := strconv.ParseFloat(toString(v), 64)
		return any(n).(T)
	case bool:
		s := strings.ToLower(toString(v))
		return any(s == "true" || s == "1").(T)
	case qtime.Date:
		return any(parseDateValue(v)).(T)
	case qtime.DateTime:
		return any(parseDateTimeValue(v)).(T)
	case []File:
		if bs, err := json.Marshal(v); err == nil {
			_ = json.Unmarshal(bs, &zero)
		}
		return zero
	}
	// 兜底：JSON 往返一次（适合自定义结构体等场景）
	if bs, err := json.Marshal(v); err == nil {
		_ = json.Unmarshal(bs, &zero)
	}
	return zero
}

// toString 把任意类型序列化为字符串，与旧 GetString 行为保持一致：
// string 直接返回；其他类型走 json.Marshal；marshal 失败时回退到 fmt.Sprintf("%v", v)
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	}
	if bs, err := json.Marshal(v); err == nil {
		return string(bs)
	}
	return fmt.Sprintf("%v", v)
}

func parseDateValue(v any) qtime.Date {
	var d qtime.Date
	js := fmt.Sprintf("{\"Time\":\"%s\"}", toString(v))
	_ = json.Unmarshal([]byte(js), &struct {
		Time qtime.Date
	}{Time: d})
	return d
}

func parseDateTimeValue(v any) qtime.DateTime {
	var t qtime.DateTime
	js := fmt.Sprintf("{\"Time\":\"%s\"}", toString(v))
	_ = json.Unmarshal([]byte(js), &struct {
		Time qtime.DateTime
	}{Time: t})
	return t
}

func (d *values) load(content []byte) error {
	var obj interface{}
	err := json.Unmarshal(content, &obj)
	if err != nil {
		return err
	}
	maps := make([]map[string]interface{}, 0)
	kind := reflect.TypeOf(obj).Kind()
	if kind == reflect.Slice {
		for _, o := range obj.([]interface{}) {
			if m, ok := o.(map[string]interface{}); ok {
				maps = append(maps, m)
			} else {
				if len(maps) == 0 {
					maps = append(maps, map[string]interface{}{"": []any{o}})
				} else {
					maps[0][""] = append(maps[0][""].([]any), o)
				}
			}
		}
	} else if kind == reflect.Map || kind == reflect.Struct {
		maps = append(maps, obj.(map[string]interface{}))
	} else {
		maps = append(maps, map[string]interface{}{"": obj})
	}
	d.InputRaw = obj
	d.InputMaps = maps
	return nil
}

func (d *values) getValue(key string) interface{} {
	if len(d.InputMaps) == 0 {
		return nil
	}
	var value interface{}
	if v, ok := d.InputMaps[0][key]; ok {
		// 如果存在
		value = v
	} else {
		str := stringy.New(key).CamelCase().ToLower()
		// 如果不存在，尝试查找
		for k, v := range d.InputMaps[0] {
			if str == stringy.New(k).CamelCase().ToLower() {
				value = v
				break
			}
		}
	}
	return value
}
