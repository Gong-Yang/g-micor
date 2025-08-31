package order

import (
	"github.com/Gong-Yang/g-micor/contract/order_contract"
)

type Service struct {
}

func (s Service) Create(req *order_contract.CreateReq, res *order_contract.CreateRes) (err error) {
	res_, err := Create(req)
	if err != nil {
		return err
	}
	*res = *res_
	return
}
