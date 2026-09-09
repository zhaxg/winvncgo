package tasks

import (
	"crypto/rand"
	"math/big"
)

// idChars 可用字符集（21个）
// 数字 0-9 + 清晰字母（排除易混淆的 B/D/G/J/M/N/Q/Y/I/L/O/F/S/Z/C/H）
const idChars = "0123456789AEKPRSTUVWX"

// idLength ID长度
const idLength = 6

// randomChars 可用于随机密码的字符集（62个）
// 数字 0-9 + 大写字母 A-Z + 小写字母 a-z
const randomChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateID() (string, error) {
	result := make([]byte, idLength)
	for i := 0; i < idLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		if err != nil {
			return "", err
		}
		result[i] = idChars[n.Int64()]
	}
	return string(result), nil
}

// generateRandomPassword 生成指定长度的随机密码（用于关闭时设置安全密码）
func generateRandomPassword(length int) (string, error) {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(randomChars))))
		if err != nil {
			return "", err
		}
		result[i] = randomChars[n.Int64()]
	}
	return string(result), nil
}
