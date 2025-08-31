package notify_contract

type SendEmailRequest struct {
	To      string
	Subject string
}
type SendEmailResponse struct {
	Message string
	Code    int
}
