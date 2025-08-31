package app

import (
	"github.com/Gong-Yang/g-micor/core/discover"
	"log/slog"
	"net"
	"net/rpc"
	"reflect"
	"strings"
)

func Run(addr string, gwAddr string, service ...any) {
	var ss []string
	for _, s := range service {
		serviceName := strings.Split(reflect.TypeOf(s).String(), ".")[0]
		slog.Info("register service", "service", serviceName)
		ss = append(ss, serviceName)
		err := rpc.RegisterName(serviceName, s)
		if err != nil {
			panic(err)
		}
	}
	err := rpc.Register(discover.ClientService{})
	if err != nil {
		panic(err)
	}
	discover.Reg(gwAddr, &discover.RegisterReq{
		Addr:    addr,
		Servers: ss,
	})
	listener, _ := net.Listen("tcp", addr)
	rpc.Accept(listener)

}
