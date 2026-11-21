package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// NewToken 生成签名 token：base64(JSON).base64(HMAC)。
// 签名同时覆盖 body 的类型名，一种类型的 token 不能被当成另一种类型使用。
func NewToken(body any, key string) (token string, err error) {
	marshal, err := json.Marshal(body)
	if err != nil {
		return
	}

	// 使用HMAC 进行签名
	var sign string = encoding(signBytes(typeName(body), marshal, key))

	// 使用压缩率较高的方式对二进制数据进行编码
	var bodyString string = encoding(marshal)

	token = bodyString + "." + sign
	return
}

// signBytes 计算 HMAC(类型名 + "\n" + body)
func signBytes(typ string, body []byte, key string) []byte {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(typ))
	h.Write([]byte("\n"))
	h.Write(body)
	return h.Sum(nil)
}

// typeName 返回去掉指针后的类型名，例如 authC.LoginUser
func typeName(v any) string {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return ""
	}
	return t.String()
}

// encoding 使用base64编码，压缩率高且URL安全（无填充等号）
func encoding(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// decoding 解码base64数据
func decoding(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

// VerifyToken 验证token并解析出原始数据，target 必须是指针，类型需与生成时一致
func VerifyToken(token string, key string, target any) error {
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
	expectedSign := encoding(signBytes(typeName(target), bodyData, key))

	if !hmac.Equal([]byte(expectedSign), []byte(signString)) {
		return errors.New("invalid token signature")
	}

	// 解析JSON数据
	err = json.Unmarshal(bodyData, target)
	if err != nil {
		return errors.New("failed to unmarshal token data")
	}

	return nil
}
