package util

import (
	"fmt"
	"regexp"
)

const mailRegex = "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

func IsEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 邮箱格式正则验证
	emailRegex := regexp.MustCompile(mailRegex)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("邮箱格式不正确")
	}

	return nil
}
