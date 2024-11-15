package qcache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type Caches[T any] struct {
	caches          *cache.Cache
	findingCallback func(key string) (T, bool)
}

// NewCaches
//
//	@Description: 创建缓存
//	@param defaultExpiration 默认有效期
//	@param cleanupInterval 清空间隔
//	@param findingCallback 缓存不存在时，主动查找回调
//	@return *Caches[T]
func NewCaches[T any](defaultExpiration, cleanupInterval time.Duration, findingCallback func(key string) (T, bool)) *Caches[T] {
	c := &Caches[T]{
		caches:          cache.New(defaultExpiration, cleanupInterval),
		findingCallback: findingCallback,
	}
	return c
}

// Set
//
//	@Description: 写入缓存，使用默认的缓存有效期
//	@param key
//	@param value
//	@param newExpiration
func (c *Caches[T]) Set(key string, value T) {
	c.caches.Set(key, value, 0)
}

// SetWithNewExpiration
//
//	@Description: 写入缓存, 使用新的缓存有效期
//	@param key
//	@param value
//	@param newExpiration
func (c *Caches[T]) SetWithNewExpiration(key string, value T, newExpiration time.Duration) {
	c.caches.Set(key, value, newExpiration)
}

// Get
//
//	@Description: 获取缓存
//	@param key
//	@return T
func (c *Caches[T]) Get(key string) (T, bool) {
	value, exist := c.caches.Get(key)
	if exist == false && c.findingCallback != nil {
		newValue, ok := c.findingCallback(key)
		if ok == true {
			c.caches.Set(key, newValue, 0)
			value = newValue
			exist = true
		}
	}
	if exist == false {
		return *new(T), false
	}
	return value.(T), exist
}

// Delete
//
//	@Description: 删除缓存
//	@param key
func (c *Caches[T]) Delete(key string) {
	c.caches.Delete(key)
}

// SaveToFile
//
//	@Description: 将缓存保存到文件
//	@param filePath 文件路径
//	@return error
func (c *Caches[T]) SaveToFile(filePath string) error {
	return c.caches.SaveFile(filePath)
}

// LoadFromFile
//
//	@Description: 从文件加载缓存
//	@param filePath 文件路径
//	@return error
func (c *Caches[T]) LoadFromFile(filePath string) error {
	return c.caches.LoadFile(filePath)
}
