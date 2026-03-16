package tools

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go-tmp/cmd/global"
	"regexp"
)

// 直接用正则去掉所有 {e} 前缀

func RemoveEncryptionPrefix(jsonStr string) string {
	// 修正正则表达式格式
	re := regexp.MustCompile(`^\{e\}(.*)`)
	return re.ReplaceAllString(jsonStr, "$1")
}

// pkcs7Padding 对数据进行 PKCS7 填充
func Pkcs7Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpadding 去除 PKCS7 填充
func Pkcs7Unpadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("invalid padding size")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:length-padding], nil
}

// encryptAES 加密
func EncryptAES(text string, outType string, key []byte) (string, error) {
	if text == "" {
		return "", fmt.Errorf("empty text")
	}

	plainBytes := []byte(text)
	plainBytes = Pkcs7Padding(plainBytes, aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, global.AESIv)
	encryptedBytes := make([]byte, len(plainBytes))
	mode.CryptBlocks(encryptedBytes, plainBytes)

	if outType == "hex" {
		return hex.EncodeToString(encryptedBytes), nil
	} else {
		return base64.StdEncoding.EncodeToString(encryptedBytes), nil
	}
}

// decryptAES 解密
func DecryptAES(encrypted string, key []byte, inType string) (string, error) {
	// 补齐16位
	if len(key) < 16 {
		padLen := 16 - (len(key) % 16)
		padding := bytes.Repeat([]byte{byte(32)}, padLen)
		key = append(key, padding...) // 使用 0 填充
	}

	//正则     /^\{e\}(.*)/
	if encrypted == "" {
		return "", fmt.Errorf("invalid encrypted string")
	}
	var encryptedBytes []byte
	var err error
	if inType == "hex" {
		encryptedBytes, err = hex.DecodeString(encrypted)
	} else {
		encryptedBytes, err = base64.StdEncoding.DecodeString(encrypted)
	}
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, global.AESIv)

	decryptedBytes := make([]byte, len(encryptedBytes))
	mode.CryptBlocks(decryptedBytes, encryptedBytes)

	decryptedBytes, err = Pkcs7Unpadding(decryptedBytes)
	if err != nil {
		return "", err
	}

	return string(decryptedBytes), nil
}
