package cryptx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

func Encrypt(plaintext string, keyStr string) (string, error) {
	key := []byte(keyStr)
	if err := validateKeyLen(key); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	blockMode := cipher.NewCBCEncrypter(block, iv)
	src := pkcs7Padding([]byte(plaintext), aes.BlockSize)
	encrypted := make([]byte, len(src))
	blockMode.CryptBlocks(encrypted, src)

	result := append(iv, encrypted...)
	return base64.StdEncoding.EncodeToString(result), nil
}

func Decrypt(ciphertextStr string, keyStr string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", fmt.Errorf("base64 解码失败: %w", err)
	}

	key := []byte(keyStr)
	if err := validateKeyLen(key); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	blockSize := aes.BlockSize
	if len(data) < blockSize*2 {
		return "", errors.New("加密数据长度不足，至少需要 32 字节（IV + 密文）")
	}

	iv := data[:blockSize]
	encrypted := data[blockSize:]

	if len(encrypted)%blockSize != 0 {
		return "", errors.New("密文长度不是块大小的整数倍")
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	origData := make([]byte, len(encrypted))
	blockMode.CryptBlocks(origData, encrypted)

	origData, err = pkcs7UnPadding(origData)
	if err != nil {
		return "", fmt.Errorf("PKCS7 去填充失败: %w", err)
	}
	return string(origData), nil
}

func validateKeyLen(key []byte) error {
	n := len(key)
	if n != 16 && n != 24 && n != 32 {
		return fmt.Errorf("AES 密钥长度必须为 16/24/32 字节，当前: %d", n)
	}
	return nil
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("数据为空")
	}

	unpadding := int(data[length-1])
	if unpadding == 0 || unpadding > aes.BlockSize || unpadding > length {
		return nil, fmt.Errorf("无效的填充值: %d", unpadding)
	}

	// 验证所有 padding 字节是否一致
	for i := length - unpadding; i < length; i++ {
		if data[i] != byte(unpadding) {
			return nil, errors.New("padding 字节不一致")
		}
	}

	return data[:(length - unpadding)], nil
}
