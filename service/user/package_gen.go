package user

import (
	"github.com/Gong-Yang/g-micor/contract/user_contract"
)

type Service struct {
}

func (s Service) Register(req *user_contract.RegisterReq, res *user_contract.RegisterRes) (err error) {
	res_, err := Register(req)
	if err != nil {
		return
	}
	*res = *res_
	return
}
