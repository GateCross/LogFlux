package cryptx

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

func Decrypt(encryptStr string, keyStr string) (string, error) {
	decodeBytes, err := base64.StdEncoding.DecodeString(encryptStr)
	if err != nil {
		return "", err
	}

	key := []byte(keyStr)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	blockMode := cipher.NewCBCDecrypter(block, key[:block.BlockSize()])
	origData := make([]byte, len(decodeBytes))
	blockMode.CryptBlocks(origData, decodeBytes)
	origData = PKCS7UnPadding(origData)
	return string(origData), nil
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
