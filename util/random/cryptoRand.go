package random

import (
	"crypto/rand"
	"encoding/hex"
)

// RandHex 快速生成相对安全的hex随机串
func RandHex() string {
	bytes := make([]byte, 16)
	ran, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes[:ran])
}
