package qf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type defCrypto struct {
	key string
}

// DefCrypto 默认解密方案
func DefCrypto(key string) ICrypto {
	return &defCrypto{key: key}
}

// Decrypt 内容解密
func (d *defCrypto) Decrypt(content string) (string, error) {
	// 先解码 base64
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %v", err)
	}

	// 先将传入的密钥串加密后，得到密钥
	hash := sha256.Sum256([]byte(d.key))

	// 再用密钥进行解密
	block, err := aes.NewCipher(hash[:])
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

	// 返回加密串
	return string(plaintext), nil
}
