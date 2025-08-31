package user_contract

import (
	"github.com/Gong-Yang/g-micor/core/discover"
	"log/slog"
)

var RegisterI func(req *RegisterReq) (res *RegisterRes, err error)

func Register(req *RegisterReq) (res *RegisterRes, err error) {
	if RegisterI != nil {
		return RegisterI(req)
	}
	//TODO RPC call
	client, err := discover.Discover("user")
	if err != nil {
		slog.Info("user discover error", "err", err)
		return
	}
	err = client.Call("user.Register", req, res)
	if err != nil { //考虑重连
		return
	}
	return
}
