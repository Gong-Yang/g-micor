package notify

import (
	"github.com/Gong-Yang/g-micor/contract/notify_contract"
	"log/slog"
)

// export
// SendEmail 发送邮件
func SendEmail(req *notify_contract.SendEmailRequest) (res *notify_contract.SendEmailResponse, err error) {
	slog.Info("send email", "req", req)
	return &notify_contract.SendEmailResponse{
		Code:    201,
		Message: "success",
	}, nil
}
