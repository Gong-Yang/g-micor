package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"google.golang.org/protobuf/proto"
)

func NewToken(body proto.Message, key string) (token string, err error) {
	marshal, err := proto.Marshal(body)
	if err != nil {
		return
	}

	// 使用HMAC 进行签名
	h := hmac.New(sha256.New, []byte(key))
	h.Write(marshal)
	var sign string = encoding(h.Sum(nil))

	// 使用压缩率较高的方式对二进制数据进行编码
	var bodyString string = encoding(marshal)

	token = bodyString + "." + sign
	return
}

// encoding 使用base64编码，压缩率高且URL安全（无填充等号）
func encoding(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// decoding 解码base64数据
func decoding(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

// VerifyToken 验证token并解析出原始数据
func VerifyToken(token string, key string, target proto.Message) error {
	// 分割token
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("invalid token format")
	}

	bodyString := parts[0]
	signString := parts[1]

	// 解码数据部分
	bodyData, err := decoding(bodyString)
	if err != nil {
		return errors.New("invalid token body encoding")
	}

	// 验证签名
	h := hmac.New(sha256.New, []byte(key))
	h.Write(bodyData)
	expectedSign := encoding(h.Sum(nil))

	if !hmac.Equal([]byte(expectedSign), []byte(signString)) {
		return errors.New("invalid token signature")
	}

	// 解析protobuf数据
	err = proto.Unmarshal(bodyData, target)
	if err != nil {
		return errors.New("failed to unmarshal token data")
	}

	return nil
}
