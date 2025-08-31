package order_contract

import (
	"github.com/Gong-Yang/g-micor/core/discover"
	"log/slog"
)

var CreateI func(user *CreateReq) (res *CreateRes, err error)

func Create(req *CreateReq) (res *CreateRes, err error) {
	if CreateI != nil {
		return CreateI(req)
	}
	//TODO RPC call
	client, err := discover.Discover("order")
	if err != nil {
		slog.Info("order discover error", "err", err)
		return
	}
	err = client.Call("order.Create", req, res)
	if err != nil { //考虑重连
		return
	}
	return
}
