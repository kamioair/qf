package qreflect

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Reflect struct {
	t      reflect.Type
	v      reflect.Value
	obj    interface{}
	kv     map[string]interface{}
	orders []string
}

//
// New
//  @Description: 创建反射实例
//  @param object 任意对象
//  @return Reflect
//
func New(object interface{}) *Reflect {
	r := &Reflect{
		t:   reflect.TypeOf(object),
		v:   reflect.ValueOf(object),
		obj: object,
	}
	// 通过json反转为字典
	r.kv = make(map[string]interface{})
	if str, err := json.Marshal(object); err == nil {
		_ = json.Unmarshal(str, &r.kv)
		// 获取属性排序
		matches := regexp.MustCompile(`"(\w+)":`).FindAllStringSubmatch(string(str), -1)
		for _, match := range matches {
			if len(match) == 2 {
				r.orders = append(r.orders, match[1])
			} else if len(match) == 1 {
				r.orders = append(r.orders, match[0])
			}
		}
	}
	return r
}

//
// IsPtr
//  @Description: 判断是否是指针
//  @receiver r
//  @return bool
//
func (r *Reflect) IsPtr() bool {
	return r.v.Kind() == reflect.Ptr
}

//
// IsMap
//  @Description: 是否是字典
//  @return bool
//
func (r *Reflect) IsMap() bool {
	if r.v.Kind() == reflect.Map {
		return true
	}
	if r.v.Kind() == reflect.Ptr {
		if r.v.Elem().Kind() == reflect.Map {
			return true
		}
	}
	return false
}

//
// IsSlice
//  @Description: 是否是切片
//  @return bool
//
func (r *Reflect) IsSlice() bool {
	if r.v.Kind() == reflect.Slice {
		return true
	}
	if r.v.Kind() == reflect.Ptr {
		if r.v.Elem().Kind() == reflect.Slice {
			return true
		}
	}
	return false
}

//
// Interface
//  @Description: 返回对象
//  @return interface{}
//
func (r *Reflect) Interface() interface{} {
	return r.obj
}

//
// InterfaceArray
//  @Description: 返回对象
//  @return interface{}
//
func (r *Reflect) InterfaceArray() []interface{} {
	if r.IsSlice() {
		v := r.v
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		list := make([]interface{}, 0)
		for i := 0; i < v.Len(); i++ {
			tp := reflect.TypeOf(v.Index(i).Interface())
			nv := reflect.New(tp)
			nv.Elem().Set(v.Index(i))
			list = append(list, nv.Interface())
		}
		return list
	}
	return nil
}

//
// Clear
//  @Description: 清空切片
//
func (r *Reflect) Clear() {
	if r.IsSlice() {
		r.v.Elem().Set(reflect.MakeSlice(r.t.Elem(), 0, 0))
	}
}

//
// ToMap
//  @Description: 转为字典
//  @return map[string]interface{}
//
func (r *Reflect) ToMap() map[string]interface{} {
	return r.kv
}

//
// ToMaps
//  @Description: 转为字典列表
//  @return []map[string]interface{}
//
func (r *Reflect) ToMaps() []map[string]interface{} {
	finals := make([]map[string]interface{}, 0)
	js, _ := json.Marshal(r.obj)
	_ = json.Unmarshal(js, &finals)
	return finals
}

//
// ToMapExpandAll
//  @Description: 转为字典，此方法会遍历所有字典值，将值为json字符串的再次展开
//  @return map[string]
//
func (r *Reflect) ToMapExpandAll() map[string]interface{} {
	final := map[string]interface{}{}
	expandAll(final, r.kv, r.orders)
	return final
}

func expandAll(source map[string]interface{}, target map[string]interface{}, targetOrders []string) {
	values := map[string]interface{}{}
	for k, v := range target {
		values[k] = v
	}
	// 先按排序执行一次
	if targetOrders != nil {
		for _, k := range targetOrders {
			doExpand(source, k, values[k])
			delete(values, k)
		}
	}
	// 然后处理剩余的
	for k, v := range values {
		doExpand(source, k, v)
	}
}

func doExpand(source map[string]interface{}, k string, v interface{}) {
	if k == "Summary" || k == "FullInfo" {
		if v == nil || v == "" {
			return
		}
	}
	// 判断值类型
	t := reflect.TypeOf(v)
	if t != nil {
		switch t.Kind() {
		case reflect.String:
			str := v.(string)
			var js map[string]interface{}
			if err := json.Unmarshal([]byte(str), &js); err != nil {
				source[k] = str
			} else {
				for k, v := range js {
					source[k] = v
				}
			}
		case reflect.Map:
			mp := map[string]interface{}{}
			expandAll(mp, v.(map[string]interface{}), nil)
			source[k] = mp
		case reflect.Slice:
			tgs := v.([]interface{})
			mps := make([]map[string]interface{}, len(tgs))
			for i, t := range tgs {
				mp := map[string]interface{}{}
				expandAll(mp, t.(map[string]interface{}), nil)
				mps[i] = mp
			}
			source[k] = mps
		default:
			source[k] = v
		}
	} else {
		source[k] = v
	}
}

//func getOrders(t reflect.Type) []string {
//	orders := make([]string, 0)
//	if t.Kind() == reflect.Ptr {
//		t = t.Elem()
//	}
//	for i := 0; i < t.NumField(); i++ {
//		orders = append(orders, t.Field(i).Name)
//	}
//	return orders
//}

//
// Get
//  @Description: 获取属性值
//  @param name 属性名
//  @return interface
//
func (r *Reflect) Get(name string) interface{} {
	return r.ToMapExpandAll()[name]
}

//
// Set
//  @Description: 设置属性值
//  @param name 属性名
//  @param value 值
//  @return error
//
func (r *Reflect) Set(name string, value interface{}) error {
	return r.SetAny(map[string]interface{}{name: value})
}

//
// SetAny
//  @Description: 将任意对象写入到对象中
//  @param mapOrStructOrSlice 字典、结构体、切片
//  @return error
//
func (r *Reflect) SetAny(mapOrStructOrSlice ...interface{}) error {
	values := mapOrStructOrSlice
	if values == nil || len(values) == 0 {
		return errors.New("the value 's length must be greater than 0")
	}
	// 如果对象不是指针，则无法执行
	if r.IsPtr() == false {
		return errors.New("the obj 's kind must be ptr")
	}
	// 如果是结构体
	kind := r.t.Elem().Kind()
	if kind == reflect.Struct {
		for _, value := range values {
			e := r.setStruct(value)
			if e != nil {
				return e
			}
		}
		return nil
	}
	// 如果是切片
	if kind == reflect.Slice {
		// 先计算长度，如果全部长度不一致，则失败
		vLen := 0
		vValue := make([]reflect.Value, 0)
		for _, value := range values {
			t := reflect.TypeOf(value)
			v := reflect.ValueOf(value)
			// 如果参数不是切片，则失败
			if t.Kind() != reflect.Slice {
				return errors.New("the value 's type must be slice")
			}
			if vLen != 0 && vLen != v.Len() {
				return errors.New("the length of the parameter is inconsistent")
			}
			vLen = v.Len()
			vValue = append(vValue, v)
		}
		slice := reflect.MakeSlice(r.t.Elem(), vLen, vLen)
		for _, value := range vValue {
			e := r.setSlice(slice, value)
			if e != nil {
				return e
			}
		}
		r.v.Elem().Set(reflect.AppendSlice(r.v.Elem(), slice))
		return nil
	}
	return errors.New("the obj 's kind must be struct or slice")
}

func (r *Reflect) setStruct(sub interface{}) error {
	subT := reflect.TypeOf(sub)
	subV := reflect.ValueOf(sub)
	// 如果入参是切片，则只取第一条
	if subT.Kind() == reflect.Slice {
		if subV.Len() == 0 {
			return errors.New("the value 's length must be greater than 0")
		}
		subT = subV.Index(0).Type()
		subV = subV.Index(0)
	}
	// 先通过json反转一次
	js, _ := json.Marshal(subV.Interface())
	_ = json.Unmarshal(js, r.obj)
	// 再进行一次赋值
	v := r.v.Elem()
	return r.set(v, subT, subV)
}

func (r *Reflect) setSlice(v reflect.Value, sub reflect.Value) error {
	for i := 0; i < sub.Len(); i++ {
		// 先通过json反转一次
		tp := reflect.TypeOf(v.Index(i).Interface())
		obj := reflect.New(tp).Interface()
		js, _ := json.Marshal(sub.Index(i).Interface())
		_ = json.Unmarshal(js, &obj)
		vv := reflect.ValueOf(obj)
		if vv.Kind() == reflect.Ptr {
			vv = vv.Elem()
		}
		v.Index(i).Set(vv)

		// 再补上其他的
		vv = v.Index(i)
		//st := sub.Index(i).Type()
		st := reflect.TypeOf(sub.Index(i).Interface())
		sv := sub.Index(i)
		e := r.set(vv, st, sv)
		if e != nil {
			return e
		}
	}
	return nil
}

func (r *Reflect) set(v reflect.Value, subT reflect.Type, subV reflect.Value) error {
	if subT.Kind() == reflect.Map {
		for _, m := range subV.MapKeys() {
			f := v.FieldByName(m.String())
			if f.IsValid() {
				if f.Kind() == reflect.Ptr && !f.IsNil() {
					f = f.Elem()
				}
			}
			if f.CanSet() {
				r.inField(f, subV.MapIndex(m).Interface())
			}
		}
	} else {
		if subT.Kind() == reflect.Ptr {
			subT = subT.Elem()
			subV = subV.Elem()
			if subV.Kind() == reflect.Ptr {
				subV = subV.Elem()
			}
		}
		for i := 0; i < subT.NumField(); i++ {
			f := v.FieldByName(subT.Field(i).Name)
			if f.IsValid() {
				if f.Kind() == reflect.Ptr && !f.IsNil() {
					f = f.Elem()
				}
			}
			if f.CanSet() {
				r.inField(f, subV.Field(i).Interface())
			}
		}
	}
	return nil
}

func (r *Reflect) inField(field reflect.Value, value interface{}) {
	tp := strings.ToLower(field.Type().Kind().String())
	vv, err := r.convert(field.Type(), value)
	if err == nil {
		// 如果是自定义类型包原生类型，则特殊处理
		if tp == "ptr" {
			ptr := reflect.New(field.Type().Elem())
			ptr.Elem().Set(reflect.ValueOf(vv))
			field.Set(ptr)
		} else if tp != field.Type().String() {
			switch tp {
			case "uint", "uint8", "uint32", "uint64":
				u, e := strconv.ParseUint(fmt.Sprintf("%v", vv), 10, 64)
				if e == nil {
					field.SetUint(u)
				}
			case "int", "int8", "int32", "int64":
				u, e := strconv.ParseInt(fmt.Sprintf("%v", vv), 10, 64)
				if e == nil {
					field.SetInt(u)
				}
			case "float32":
				u, e := strconv.ParseFloat(fmt.Sprintf("%v", vv), 32)
				if e == nil {
					field.SetFloat(u)
				}
			case "float64":
				u, e := strconv.ParseFloat(fmt.Sprintf("%v", vv), 64)
				if e == nil {
					field.SetFloat(u)
				}
			case "bool":
				u, e := strconv.ParseBool(fmt.Sprintf("%v", vv))
				if e == nil {
					field.SetBool(u)
				}
			}
		} else {
			field.Set(reflect.ValueOf(vv))
		}
	}
}

// 类型转换
func (r *Reflect) convert(tp reflect.Type, value interface{}) (interface{}, error) {
	typeName := strings.ToLower(tp.Kind().String())
	if tp.Kind() == reflect.Ptr {
		typeName = strings.ToLower(tp.Elem().Kind().String())
	}
	str := fmt.Sprintf("%v", value)
	switch typeName {
	case "string":
		return str, nil
	case "int":
		return strconv.Atoi(str)
	case "int64":
		return strconv.ParseInt(str, 10, 64)
	case "int32":
		v, e := strconv.ParseInt(str, 10, 32)
		return int32(v), e
	case "int16":
		v, e := strconv.ParseInt(str, 10, 16)
		return int16(v), e
	case "int8":
		v, e := strconv.ParseInt(str, 10, 8)
		return int8(v), e
	case "uint":
		v, e := strconv.ParseUint(str, 10, 64)
		return uint(v), e
	case "uint64":
		return strconv.ParseUint(str, 10, 64)
	case "uint32":
		v, e := strconv.ParseUint(str, 10, 32)
		return uint32(v), e
	case "uint16":
		v, e := strconv.ParseUint(str, 10, 16)
		return uint16(v), e
	case "uint8":
		v, e := strconv.ParseUint(str, 10, 8)
		return uint8(v), e
	case "bool":
		return strconv.ParseBool(str)
	case "float64":
		return strconv.ParseFloat(str, 64)
	case "float32":
		v, e := strconv.ParseFloat(str, 32)
		return float32(v), e
	case "time.Time":
		return value, nil
	case "time.Duration":
		v, e := strconv.Atoi(str)
		return time.Duration(v) * time.Millisecond, e
	case "*string":
		return &str, nil
	case "*int":
		v, e := strconv.Atoi(str)
		return &v, e
	case "*int64":
		v, e := strconv.ParseInt(str, 10, 64)
		return &v, e
	case "*int32":
		v, e := strconv.ParseInt(str, 10, 32)
		o := int32(v)
		return &o, e
	case "*int16":
		v, e := strconv.ParseInt(str, 10, 16)
		o := int16(v)
		return &o, e
	case "*int8":
		v, e := strconv.ParseInt(str, 10, 8)
		o := int8(v)
		return &o, e
	case "*uint":
		v, e := strconv.ParseUint(str, 10, 64)
		o := uint(v)
		return &o, e
	case "*uint64":
		v, e := strconv.ParseUint(str, 10, 64)
		return &v, e
	case "*uint32":
		v, e := strconv.ParseUint(str, 10, 32)
		o := uint32(v)
		return &o, e
	case "*uint16":
		v, e := strconv.ParseUint(str, 10, 16)
		o := uint16(v)
		return &o, e
	case "*uint8":
		v, e := strconv.ParseUint(str, 10, 8)
		o := uint8(v)
		return &o, e
	case "*bool":
		v, e := strconv.ParseBool(str)
		return &v, e
	case "*float64":
		v, e := strconv.ParseFloat(str, 64)
		return &v, e
	case "*float32":
		v, e := strconv.ParseFloat(str, 32)
		o := float32(v)
		return &o, e
	case "*time.Time":
		return &value, nil
	case "*time.Duration":
		v, e := strconv.Atoi(str)
		o := time.Duration(v) * time.Millisecond
		return &o, e
	}
	return nil, errors.New("Not support Type " + typeName)
}
