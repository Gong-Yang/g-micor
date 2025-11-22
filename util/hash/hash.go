package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func HMACSha256(body, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}
