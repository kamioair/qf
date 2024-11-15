package qconvert

import (
	"encoding/json"
	"errors"
	"github.com/kamioair/qf/utils/qreflect"
	"strings"
)

var DB convertDatabase

type convertDatabase struct {
}

// ToDbModel
//
//	@Description: 将Api的Model序列化为数据库的Model
//	@param apiModel ApiModel
//	@param refDbModel 待转换的DbModel
func (c convertDatabase) ToDbModel(apiModel any, refDbModel any) {
	// 先将apiModel转为字典
	js, _ := json.Marshal(apiModel)
	values := map[string]interface{}{}
	_ = json.Unmarshal(js, &values)
	// 在写入到数据库Model
	_ = c.setModel(refDbModel, values)
}

// ToApiModel
//
//	@Description: 将数据库的Model反序列化到Api的Model
//	@param dbModel DbModel
//	@param refApiModel 待转换的ApiModel
func (c convertDatabase) ToApiModel(dbModel any, refApiModel any) {
	api := qreflect.New(refApiModel)
	// 将数据库Model中的内容写入到apiModel中
	_ = api.SetAny(qreflect.New(dbModel).ToMapExpandAll())
}

func (c convertDatabase) setModel(objectPtr interface{}, value map[string]interface{}) error {
	if objectPtr == nil {
		return errors.New("the object cannot be empty")
	}
	ref := qreflect.New(objectPtr)
	// 必须为指针
	if ref.IsPtr() == false {
		return errors.New("the object must be pointer")
	}

	// 修改外部值
	if value != nil {
		e := ref.SetAny(value)
		if e != nil {
			return e
		}
	}
	// 修改Info
	return c.setInfo(ref, value)
}

func (c convertDatabase) setInfo(ref *qreflect.Reflect, value map[string]interface{}) error {
	all := ref.ToMap()

	// 复制一份
	temp := map[string]interface{}{}
	for k, v := range value {
		temp[k] = v
	}

	// 转摘要
	if field, ok := temp["SummaryFields"]; ok && field != "" {
		e := ref.Set("Summary", c.fields(field, all["Summary"], all, &temp))
		if e != nil {
			return e
		}
	}
	// 转信息
	if field, ok := temp["InfoFields"]; ok && field != "" {
		e := ref.Set("FullInfo", c.fields(field, all["FullInfo"], all, &temp))
		if e != nil {
			return e
		}
		return nil
	}

	// 将剩余的全部写入到Info中
	if info, ok := all["FullInfo"]; ok {
		mp := map[string]interface{}{}
		_ = json.Unmarshal([]byte(info.(string)), &mp)
		for k, v := range temp {
			if k == "SummaryFields" || k == "InfoFields" {
				continue
			}
			if _, ok := all[k]; ok == false {
				mp[k] = v
			}
		}
		mj, _ := json.Marshal(mp)
		e := ref.Set("FullInfo", string(mj))
		if e != nil {
			return e
		}
	}
	return nil
}

func (c convertDatabase) fields(field interface{}, source interface{}, all map[string]interface{}, values *map[string]interface{}) string {
	if field == nil || field.(string) == "" {
		return ""
	}
	// 获取原始数据并转为字典
	mp := map[string]interface{}{}
	if source != nil {
		_ = json.Unmarshal([]byte(source.(string)), &mp)
	}
	// 获取需要的值
	temp := *values
	for _, name := range strings.Split(field.(string), ",") {
		if _, ok := all[name]; ok == false {
			if _, ok2 := temp[name]; ok2 {
				mp[name] = temp[name]
				delete(temp, name)
			}
		}
	}
	values = &temp
	// 返回
	if len(mp) == 0 {
		return ""
	}
	mj, _ := json.Marshal(mp)
	return string(mj)
}
