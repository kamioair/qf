package qdefine

import (
	"errors"
	"github.com/kamioair/qf/utils/qreflect"
	"gorm.io/gorm"
	"reflect"
	"time"
)

type DbSimple struct {
	Id       uint64   `gorm:"primaryKey"` // 唯一号
	LastTime DateTime `gorm:"index"`      // 最后操作时间时间
}

type DbFull struct {
	Id       uint64   `gorm:"primaryKey"` // 唯一号
	LastTime DateTime `gorm:"index"`      // 最后操作时间时间
	Summary  string   // 摘要
	FullInfo string   // 其他扩展内容
}

type BaseDao[T any] struct {
	db *gorm.DB
}

// NewDao
//
//	@Description: 创建Dao
//	@param db
//	@return *BaseDao[T]
func NewDao[T any](db *gorm.DB) *BaseDao[T] {
	// 主动创建数据库
	m := new(T)
	name := reflect.TypeOf(*m).Name()
	if db.Migrator().HasTable(name) == false {
		err := db.AutoMigrate(m)
		if err != nil {
			return nil
		}
	}
	return &BaseDao[T]{db: db}
}

// DB
//
//	@Description: 返回数据库
//	@return *gorm.DB
func (dao *BaseDao[T]) DB() *gorm.DB {
	return dao.db
}

// Create
//
//	@Description: 新建一条记录
//	@param model 对象
//	@return bool
//	@return error
func (dao *BaseDao[T]) Create(model *T) error {
	ref := qreflect.New(model)
	if ref.Get("LastTime") == "0001-01-01 00:00:00" {
		_ = ref.Set("LastTime", NewDateTime(time.Now()))
	}
	// 提交
	result := dao.DB().Create(model)
	return result.Error
}

// CreateList
//
//	@Description: 创建一组列表
//	@param list
//	@return error
func (dao *BaseDao[T]) CreateList(list []T) error {
	// 启动事务创建
	err := dao.DB().Transaction(func(tx *gorm.DB) error {
		for _, model := range list {
			ref := qreflect.New(model)
			if ref.Get("LastTime") == "0001-01-01 00:00:00" {
				_ = ref.Set("LastTime", NewDateTime(time.Now()))
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// Update
//
//	@Description: 修改一条记录
//	@param model 对象
//	@return bool
//	@return error
func (dao *BaseDao[T]) Update(model *T) error {
	ref := qreflect.New(model)
	if ref.Get("LastTime") == "0001-01-01 00:00:00" {
		_ = ref.Set("LastTime", NewDateTime(time.Now()))
	}
	// 提交
	result := dao.DB().Model(model).Updates(model)
	if result.RowsAffected > 0 {
		return nil
	}
	if result.Error != nil {
		return result.Error
	}
	return errors.New("update record does not exist")
}

// Save
//
//	@Description: 修改一条记录（不存在则新增）
//	@param model 对象
//	@return bool
//	@return error
func (dao *BaseDao[T]) Save(model *T) error {
	ref := qreflect.New(model)
	if ref.Get("LastTime") == "0001-01-01 00:00:00" {
		_ = ref.Set("LastTime", NewDateTime(time.Now()))
	}
	// 提交
	result := dao.DB().Save(model)
	return result.Error
}

// Delete
//
//	@Description: 删除一条记录
//	@param id 记录Id
//	@return bool
//	@return error
func (dao *BaseDao[T]) Delete(id uint64) error {
	result := dao.DB().Where("id = ?", id).Delete(new(T))
	return result.Error
}

// DeleteCondition
//
//	@Description: 自定义条件删除数据
//	@param condition
//	@param args
//	@return error
func (dao *BaseDao[T]) DeleteCondition(condition string, args ...any) error {
	result := dao.DB().Where(condition, args...).Delete(new(T))
	return result.Error
}

// GetModel
//
//	@Description: 获取一条记录
//	@param id
//	@return *T
//	@return error
func (dao *BaseDao[T]) GetModel(id uint64) (*T, error) {
	// 创建空对象
	model := new(T)
	// 查询
	result := dao.DB().Where("id = ?", id).Find(model)
	// 如果异常或者未查询到任何数据
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return model, nil
}

// CheckExist
//
//	@Description: 验证数据是否存在
//	@param id
//	@return bool
//	@return error
func (dao *BaseDao[T]) CheckExist(id uint64) bool {
	// 创建空对象
	model := new(T)
	// 查询
	result := dao.DB().Where("id = ?", id).Find(model)
	// 如果异常或者未查询到任何数据
	if result.Error != nil || result.RowsAffected == 0 {
		return false
	}
	return true
}

// GetList
//
//	@Description: 查询一组列表
//	@param startId 起始Id
//	@param maxCount 最大数量
//	@return []*T
//	@return error
func (dao *BaseDao[T]) GetList(startId uint64, maxCount int) ([]T, error) {
	list := make([]T, 0)
	// 查询
	result := dao.DB().Limit(int(maxCount)).Offset(int(startId)).Find(&list)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return list, nil
}

// GetAll
//
//	@Description: 返回所有列表
//	@return []*T
//	@return error
func (dao *BaseDao[T]) GetAll() ([]T, error) {
	list := make([]T, 0)
	// 查询
	result := dao.DB().Find(&list)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return list, nil
}

// GetCondition
//
//	@Description: 条件查询一条记录
//	@param query 条件，如 id = ? 或 id IN (?) 等
//	@param args 条件参数，如 id, ids 等
//	@return []*T
//	@return error
func (dao *BaseDao[T]) GetCondition(query interface{}, args ...interface{}) (*T, error) {
	model := new(T)
	// 查询
	result := dao.DB().Where(query, args...).Find(model)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return model, nil
}

// GetConditions
//
//	@Description: 条件查询一组列表
//	@param query 条件，如 id = ? 或 id IN (?) 等
//	@param args 条件参数，如 id, ids 等
//	@return []*T
//	@return error
func (dao *BaseDao[T]) GetConditions(query interface{}, args ...interface{}) ([]T, error) {
	list := make([]T, 0)
	// 查询
	result := dao.DB().Where(query, args...).Find(&list)
	if result.Error != nil || result.RowsAffected == 0 {
		return nil, result.Error
	}
	return list, nil
}

// GetConditionsLimit
//
//	@Description: 条件查询一组列表
//	@param maxCount 最大数量
//	@param query 条件，如 id = ? 或 id IN (?) 等
//	@param args 条件参数，如 id, ids 等
//	@return []*T
//	@return error
func (dao *BaseDao[T]) GetConditionsLimit(maxCount int, query interface{}, args ...interface{}) ([]*T, error) {
	list := make([]*T, 0)
	// 查询
	if maxCount > 0 {
		result := dao.DB().Where(query, args...).Limit(maxCount).Find(&list)
		if result.Error != nil || result.RowsAffected == 0 {
			return nil, result.Error
		}
	} else {
		result := dao.DB().Where(query, args...).Find(&list)
		if result.Error != nil || result.RowsAffected == 0 {
			return nil, result.Error
		}
	}
	return list, nil
}

// GetCount
//
//	@Description: 获取总记录数
//	@return int64
func (dao *BaseDao[T]) GetCount() int64 {
	// 创建空对象
	model := new(T)
	// 查询
	var count int64
	dao.DB().Model(model).Count(&count)
	return count
}
