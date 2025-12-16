package qf

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// IModule 模块入口接口
type IModule interface {
	// Run 同步运行模块，执行后会等待直到程序退出，单进程仅单模块时使用（exe模式）
	Run()
	// RunAsync 异步运行模块，执行后不等待，单进程需要启动多模块时使用（dll模式）
	RunAsync()
	// Stop 停止模块
	Stop()
}

// IService 模块功能接口
type IService interface {
	Reg(reg *Reg)     // 注册事件
	GetInvokes() *Reg // 返回注册事件

	SendLogDebug(content string)            // 调试日志
	SendLogWarn(content string)             // 警告日志
	SendLogError(content string, err error) // 错误日志

	// 内部使用的方法
	setEnv(reg *Reg, adapter easyCon.IAdapter, config *Config, callback CallbackDelegate)
}

// IConfig 配置接口
type IConfig interface {
	getBaseConfig() *Config
}

// ICrypto 加解密接口
type ICrypto interface {
	// Decrypt 加密
	Decrypt(content string) (string, error)
}

type CallbackReq struct {
	PType   easyCon.EPType
	ReqTime string
	Route   string
	Content string
}

var instance IService

// LogDebug 发送Debug日志
func LogDebug(content string) {
	if instance == nil {
		return
	}

	instance.SendLogDebug(content)
}

// LogWarn 发送Warn日志
func LogWarn(content string) {
	if instance == nil {
		return
	}

	instance.SendLogWarn(content)
}

// LogError 发送Error日志
func LogError(content string, err error) {
	if instance == nil {
		return
	}

	instance.SendLogError(content, err)
}

type defCrypto struct {
	key string
}

func DefCrypto(key string) ICrypto {
	return &defCrypto{key: key}
}

func (d *defCrypto) Decrypt(content string) (string, error) {
	// 先解码 base64
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	block, err := aes.NewCipher([]byte(d.key))
	if err != nil {
		return "", fmt.Errorf("cipher creation failed: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM creation failed: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("content too short")
	}

	// 正确分离 nonce 和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %v", err)
	}

	return string(plaintext), nil
}
