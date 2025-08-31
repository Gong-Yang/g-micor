package discover

import "errors"

type (
	// 服务发现请求
	Req struct {
		Addr   string // 请求者的地址
		Server string // 请求者需要发现的服务
	}
	Resp struct {
		Server string   // 服务名称
		Addr   []string // 服务地址
	}

	// 服务注册请求
	RegisterRes struct {
		Code int
	}
	RegisterReq struct {
		Addr    string
		Name    string
		Servers []string
	}

	// ping 请求
	PingReq struct{}
	PingRes struct{}

	// 服务新增通知
	NotifyReq struct {
		Type   string // 通知类型 1. register 服务上线 2. del 服务下线
		Server string // 服务名
		Addr   string // 服务地址
	}
	NotifyRes struct {
	}
)

var (
	ErrServerNotFind = errors.New("ErrServerNotFind")
)
