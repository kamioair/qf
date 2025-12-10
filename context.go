package qf

import (
	"encoding/json"
	"fmt"
	"github.com/gobeam/stringy"
	"github.com/kamioair/utils/qtime"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"reflect"
	"strconv"
	"strings"
)

// File 文件
type File struct {
	Name string // 文件名
	Size int64  // 文件大小
	Data []byte // 内容
}

type CommPack struct {
	Id   uint64
	From string
	To   string
}

// IContext 上下文
type IContext interface {
	GetString(key string) string
	GetInt(key string) int
	GetUInt(key string) uint64
	GetByte(key string) byte
	GetBool(key string) bool
	GetDate(key string) qtime.Date
	GetDateTime(key string) qtime.DateTime
	GetFiles(key string) []File
	GetStruct(key string, refStruct any)
	GetCommPack() CommPack
	Raw() any
	Json() string
}

type context struct {
	values *values
	pack   CommPack
}

type values struct {
	InputMaps   []map[string]interface{}
	InputRaw    interface{}
	OutputValue interface{}
}

func NewContent(value any, reqPack *easyCon.PackReq, noticePack *easyCon.PackNotice) (IContext, error) {
	ctx := &context{
		values: &values{
			InputMaps: make([]map[string]interface{}, 0),
		},
		pack: CommPack{},
	}
	err := setData(ctx, value)
	if err != nil {
		return nil, err
	}
	if reqPack != nil {
		ctx.pack.Id = reqPack.Id
		ctx.pack.From = reqPack.From
		ctx.pack.To = reqPack.To
	}
	if noticePack != nil {
		ctx.pack.Id = noticePack.Id
		ctx.pack.From = noticePack.From
	}
	return ctx, nil
}

func setData(ctx *context, data any) error {
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

func (c *context) GetString(key string) string {
	value := c.values.getValue(key)
	// 返回
	if value == nil {
		return ""
	}
	str := ""
	switch value.(type) {
	case string:
		str = fmt.Sprintf("%s", value)
	default:
		temp, err := json.Marshal(value)
		if err != nil {
			str = fmt.Sprintf("%v", value)
		} else {
			str = string(temp)
		}
	}
	return str
}

func (c *context) GetInt(key string) int {
	num, err := strconv.Atoi(c.GetString(key))
	if err != nil {
		panic(err)
	}
	return num
}

func (c *context) GetUInt(key string) uint64 {
	num, err := strconv.ParseUint(c.GetString(key), 10, 64)
	if err != nil {
		panic(err)
	}
	return num
}

func (c *context) GetByte(key string) byte {
	num, err := strconv.ParseInt(c.GetString(key), 10, 8)
	if err != nil {
		panic(err)
	}
	return byte(num)
}

func (c *context) GetBool(key string) bool {
	value := strings.ToLower(c.GetString(key))
	if value == "true" || value == "1" {
		return true
	}
	return false
}

func (c *context) GetDate(key string) qtime.Date {
	model := struct {
		Time qtime.Date
	}{}
	js := fmt.Sprintf("{\"Time\":\"%s\"}", c.GetString(key))
	err := json.Unmarshal([]byte(js), &model)
	if err != nil {
		panic(err)
	}
	return model.Time
}

func (c *context) GetDateTime(key string) qtime.DateTime {
	model := struct {
		Time qtime.DateTime
	}{}
	js := fmt.Sprintf("{\"Time\":\"%s\"}", c.GetString(key))
	err := json.Unmarshal([]byte(js), &model)
	if err != nil {
		panic(err)
	}
	return model.Time
}

func (c *context) GetFiles(key string) []File {
	value := c.values.getValue(key)
	// 返回
	if files, ok := value.([]File); ok {
		return files
	}
	return nil
}

func (c *context) GetStruct(key string, refStruct any) {
	var val any

	t := reflect.ValueOf(refStruct)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		val = c.values.InputMaps
	} else {
		val = c.values.getValue(key)
	}

	// 先转为json
	js, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	// 再反转
	err = json.Unmarshal(js, refStruct)
	if err != nil {
		panic(err)
	}
}

func (c *context) GetCommPack() CommPack {
	return c.pack
}

func (c *context) Raw() any {
	return c.values.InputRaw
}

func (c *context) Json() string {
	js, err := json.Marshal(c.values.InputRaw)
	if err != nil {
		panic(err)
	}
	return string(js)
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
			//maps = append(maps, o.(map[string]interface{}))
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
