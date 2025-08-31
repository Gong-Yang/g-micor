package discover

import (
	"errors"
	"github.com/Gong-Yang/g-micor/core/rpcx"
	"github.com/Gong-Yang/g-micor/core/syncx"
	"log/slog"
	"net/rpc"
	"time"
)

var discoverClient *rpc.Client
var localAddr string

// Reg 业务调用的服务注册
func Reg(gatewayAddr string, req *RegisterReq) {
	go func() {
		time.Sleep(time.Second)
		var res RegisterRes
		client_, err := rpc.Dial("tcp", gatewayAddr)
		if err != nil {
			panic(err)
		}
		discoverClient = client_
		err = discoverClient.Call("Service.Register", req, &res)
		if err != nil {
			panic(err)
			return
		}

		localAddr = req.Addr
		return
	}()
}

// Discover 发现服务地址
func Discover(server string) (c *rpcx.Client, err error) {
	c, err = serverManager.GetResource(server, func() (res *rpcx.Client, err error) {
		req := &Req{
			Addr:   localAddr,
			Server: server,
		}
		var resp Resp
		err = discoverClient.Call("Service.Discover", req, &resp)
		if err != nil {
			slog.Info("discover error", "server", server, "err", err)
			return nil, err
		}
		res, err = rpcx.NewClient(resp.Addr)
		if err != nil {
			slog.Info("dial server error", "server", server, "err", err)
			return nil, err
		}
		return
	})
	return
}

var serverManager = *syncx.NewResourceManager[rpcx.Client]()

type ClientService struct {
}

// SubscribeServerRegister 客户端订阅的服务发生了注册或者下线
func (c ClientService) SubscribeServerRegister(req *NotifyReq, res *NotifyRes) error {
	resource, err := serverManager.GetResource(req.Server, func() (*rpcx.Client, error) {
		return nil, errors.New("server not find")
	})
	if err != nil {
		slog.Error("TODO")
		return nil
	}
	resource.Dial(req.Addr)
	slog.Info("TODO")
	return nil
}

func (c ClientService) Ping(req *PingReq, res *PingRes) error {
	return nil
}
