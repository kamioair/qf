package qconvert

import "encoding/json"

// ToAny
//
//	@Description: 将任意类型转为指定类型，此方法如果发生会抛出
//	@param raw 原始对象
//	@return T 指定类型
func ToAny[T any](raw any) T {
	if raw == nil {
		return *new(T)
	}
	js, err := json.Marshal(raw)
	if err != nil {
		panic(err)
	}
	dbModel := new(T)
	err = json.Unmarshal(js, &dbModel)
	if err != nil {
		panic(err)
	}
	return *dbModel
}

// ToAnyError
//
//	@Description: 将任意类型转为指定类型，此方法如果发生会返回
//	@param raw 原始对象
//	@return T, error 指定类型, 异常
func ToAnyError[T any](raw any) (T, error) {
	if raw == nil {
		return *new(T), nil
	}
	js, err := json.Marshal(raw)
	if err != nil {
		return *new(T), err
	}
	dbModel := new(T)
	err = json.Unmarshal(js, &dbModel)
	if err != nil {
		return *new(T), err
	}
	return *dbModel, nil
}
