package notify

import (
	"golang_test/micorservice/contract"
	"testing"
)

func TestSendEmail(t *testing.T) {
	res := &contract.SendEmailResponse{}
	err := Service{}.SendEmail(&contract.SendEmailRequest{
		Subject: "test",
		To:      "<EMAIL>",
	}, res)
	if err != nil {
		panic(err)
	}
	t.Log(res)
}
