package tasks

import (
	"crypto/rand"
	"math/big"
)

// preGeneratedIDs 可用 ID 列表（与 vnc_des_test.go 中的加密对照表配对）
var preGeneratedIDs = []string{
	"114514", "823456", "773890", "226100", "551789",
	"330264", "998271", "664038", "109675", "447231",
}

func generateID() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(preGeneratedIDs))))
	if err != nil {
		return "", err
	}
	return preGeneratedIDs[n.Int64()], nil
}
