package random

import (
	"github.com/google/uuid"
	"strings"
)

// 定义可用于编码的字符集 - 包含数字、大小写字母和一些安全的特殊字符
const (
	// Base62字符集（数字 + 大小写字母）
	base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// ShortUUID 生成更短的UUID（16字符）
// 使用用户自定义的Base62编码
func ShortUUID() string {
	// 生成UUID
	u := uuid.New()
	// 转换为字节数组
	bytes := u[:]

	// 仅使用UUID前10个字节（80位）来生成16字符的ID
	// 这会降低唯一性，但在大多数场景下依然足够
	shortened := bytes[:10]

	// 使用Base62编码
	var result strings.Builder
	result.Grow(16) // 预分配空间

	// 手动Base62编码
	for _, b := range shortened {
		// 每个字节表示两个Base62字符
		idx1 := b % 62
		idx2 := b / 62
		result.WriteByte(base62Chars[idx1])
		result.WriteByte(base62Chars[idx2])
	}

	return result.String()[:16]
}
