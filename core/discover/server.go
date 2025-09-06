package discover

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

var (
	ErrServerNotFind = errors.New("ErrServerNotFind")
)

// Run 启动服务发现服务器
// addr: 监听地址，格式为 "host:port"
// 返回错误信息，如果启动失败
func Run(addr string) error {
	slog.Info("启动服务发现服务器", "地址", addr)

	// 监听TCP连接
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("监听地址失败", "地址", addr, "错误", err)
		return err
	}
	slog.Info("服务发现服务器监听成功", "地址", addr)
	s := grpc.NewServer()

	// 创建服务实例
	service := &Service{
		lock:          &sync.RWMutex{},
		sNameToSAddr:  make(map[string][]string),
		sAddrToSNames: make(map[string][]string),
		sNameToDAddr:  make(map[string][]string),
		addrStore:     make(map[string]ClientClient),
	}

	// 注册RPC服务
	RegisterRegisterServer(s, service)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Serve(listener)
		if err != nil {
			slog.Error("注册RPC服务失败", "错误", err)
			panic(err)
		}
	}()
	slog.Info("RegisterCenter start")

	// 启动心跳检查
	go service.startHealthCheck()
	wg.Wait()
	return nil
}

// Service 服务发现核心服务
// 负责服务的注册、发现、健康检查和通知机制
type Service struct {
	// 读写锁，保护并发访问
	lock *sync.RWMutex

	// sNameToSAddr 服务名到服务地址的映射
	// key: 服务名称，value: 提供该服务的服务器地址列表
	sNameToSAddr map[string][]string

	// sAddrToSNames 服务地址到服务名的映射
	// key: 服务器地址，value: 该服务器提供的服务名列表
	sAddrToSNames map[string][]string

	// sNameToDAddr 服务名到发现者地址的映射
	// key: 被发现的服务名，value: 请求发现该服务的客户端地址列表
	sNameToDAddr map[string][]string

	// addrStore 地址到RPC客户端的映射
	// key: 服务器地址，value: 与该服务器的RPC连接客户端
	addrStore map[string]ClientClient
	UnimplementedRegisterServer
}

// Register 处理服务注册请求
// 当一个服务启动时，会调用此方法将自己注册到服务发现中心
func (s *Service) Register(ctx context.Context, req *RegisterReq) (res *RegisterRes, err error) {
	// 获取客户端地址
	clientAddr := req.Port
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("无法获取对等方信息")
	}
	if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
		clientIP := tcpAddr.IP.String()
		clientAddr = fmt.Sprintf("[%s]%s", clientIP, clientAddr)
	}
	slog.Info("收到服务注册请求", "地址", clientAddr, "提供服务", req.Servers)

	// 建立与注册服务的RPC连接，用于后续的健康检查和通知
	conn, err := grpc.NewClient(clientAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	client := NewClientClient(conn)
	slog.Info("与注册服务建立连接成功", "服务地址", clientAddr)

	// 获取写锁，更新服务映射关系
	s.lock.Lock()
	s.addrStore[clientAddr] = client

	// 更新服务名到地址的映射
	for _, serverName := range req.Servers {
		// 检查是否已存在该地址，避免重复添加
		addrs := s.sNameToSAddr[serverName]
		alreadyExists := false
		for _, addr := range addrs {
			if addr == clientAddr {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			s.sNameToSAddr[serverName] = append(s.sNameToSAddr[serverName], clientAddr)
		}
	}

	// 更新地址到服务名的映射
	s.sAddrToSNames[clientAddr] = req.Servers

	// 释放写锁
	s.lock.Unlock()
	slog.Info("服务注册信息更新完成", "服务地址", clientAddr)

	// 通知所有关注这些服务的客户端
	for _, serverName := range req.Servers {
		s.notifySubscribers(&NotifyReq{
			Type:   "register",
			Server: serverName,
			Addr:   clientAddr,
		})
	}

	// 返回成功响应
	slog.Info("服务注册成功", "地址", clientAddr, "服务名", req.Servers)
	return
}

// Discover 处理服务发现请求
// 当一个服务需要调用另一个服务时，会调用此方法获取目标服务的地址列表
func (s *Service) Discover(ctx context.Context, req *Req) (res *Resp, err error) {
	// 获取客户端地址
	clientAddr := req.Port
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("无法获取对等方信息")
	}
	if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
		clientIP := tcpAddr.IP.String()
		clientAddr = fmt.Sprintf("[%s]%s", clientIP, clientAddr)
	}
	slog.Info("收到服务发现请求", "请求者地址", clientAddr, "目标服务", req.Server)

	// 获取读锁，查找目标服务的地址列表
	s.lock.RLock()
	sAddr := s.sNameToSAddr[req.Server]
	s.lock.RUnlock()

	// 检查服务是否存在
	if len(sAddr) == 0 {
		slog.Error("未找到目标服务", "服务名", req.Server)
		return res, ErrServerNotFind
	}

	// 获取写锁，将请求者添加到服务订阅列表
	// 这样当目标服务有变化时（新增实例、实例下线等），可以主动通知请求者
	s.lock.Lock()

	// 检查是否已经订阅，避免重复添加
	subscribers := s.sNameToDAddr[req.Server]
	alreadySubscribed := false
	for _, subscriber := range subscribers {
		if subscriber == clientAddr {
			alreadySubscribed = true
			break
		}
	}

	if !alreadySubscribed {
		s.sNameToDAddr[req.Server] = append(s.sNameToDAddr[req.Server], clientAddr)
		slog.Info("添加服务订阅者", "服务名", req.Server, "订阅者地址", clientAddr)
	}

	s.lock.Unlock()

	// 构造响应
	res = &Resp{
		Server: req.Server,
		Addr:   make([]string, len(sAddr)),
	}
	copy(res.Addr, sAddr)

	slog.Info("服务发现成功", "服务名", req.Server, "地址列表", strings.Join(sAddr, ","))
	return
}

/*
服务发现流程说明：

正常流程：
1. 注册中心启动，所有数据结构为空
2. 业务服务启动（如 user 服务）：
   - 调用 Register 方法注册服务
   - 更新 sNameToSAddr、sAddrToSNames、addrStore
   - 此时 sNameToDAddr 为空，不触发通知

3. 其他业务服务启动（如 notify 服务）：
   - 同样注册到注册中心
   - 更新相关映射表

4. 服务发现请求：
   - user 服务需要调用 notify 服务
   - 调用 Discover 方法获取 notify 服务地址
   - 将 user 服务地址添加到 sNameToDAddr 中

5. 服务扩容：
   - 新的 notify 服务实例启动
   - 注册时发现已有客户端关注此服务
   - 主动推送新的服务地址列表给关注的客户端

6. 健康检查：
   - 定期检查 addrStore 中的连接状态
   - 发现故障服务时，清理相关映射关系
   - 通知关注的客户端更新服务列表
*/

// notifySubscribers 通知所有订阅指定服务的客户端
// 当服务有变更时（新增实例、实例下线等），调用此方法通知关注的客户端
func (s *Service) notifySubscribers(notifyReq *NotifyReq) {
	serverName := notifyReq.Server
	// 获取读锁，查找订阅者列表
	s.lock.RLock()
	subscribers := s.sNameToDAddr[serverName]
	s.lock.RUnlock()

	if len(subscribers) == 0 {
		slog.Debug("无订阅者需要通知", "服务名", serverName)
		return
	}

	slog.Info("开始通知订阅者", "服务名", serverName, "订阅者数量", len(subscribers))

	// 遍历所有订阅者，发送更新通知
	for _, subscriberAddr := range subscribers {
		// 获取订阅者的RPC客户端
		s.lock.RLock()
		subscriberClient := s.addrStore[subscriberAddr]
		s.lock.RUnlock()

		if subscriberClient == nil {
			slog.Error("订阅者RPC客户端不存在", "订阅者地址", subscriberAddr)
			// 清理无效的订阅者
			s.removeSubscriber(serverName, subscriberAddr)
			continue
		}

		// 异步发送通知，避免阻塞其他操作
		go func(client ClientClient, addr string, req *NotifyReq) {
			_, err := client.SubscribeServerRegister(context.Background(), req)
			if err != nil {
				slog.Error("通知订阅者失败", "错误", err, "订阅者地址", addr, "服务名", req.Server)
				// 如果通知失败，可能是订阅者已经下线，清理订阅关系
				s.removeSubscriber(serverName, addr)
			} else {
				slog.Info("通知订阅者成功", "订阅者地址", addr, "服务名", req.Server)
			}
		}(subscriberClient, subscriberAddr, notifyReq)
	}
}

// removeSubscriber 从订阅列表中移除指定的订阅者
func (s *Service) removeSubscriber(serverName, subscriberAddr string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	subscribers := s.sNameToDAddr[serverName]
	for i, addr := range subscribers {
		if addr == subscriberAddr {
			// 移除订阅者
			s.sNameToDAddr[serverName] = append(subscribers[:i], subscribers[i+1:]...)
			slog.Info("移除无效订阅者", "服务名", serverName, "订阅者地址", subscriberAddr)
			break
		}
	}
}

// startHealthCheck 启动健康检查协程
// 定期检查所有注册服务的健康状态，清理不可用的服务
func (s *Service) startHealthCheck() {
	slog.Info("启动健康检查服务")

	// 每30秒执行一次健康检查
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *Service) performHealthCheck() {
	//slog.Info("开始执行健康检查")

	// 获取所有需要检查的地址
	s.lock.RLock()
	addrsToCheck := make([]string, 0, len(s.addrStore))
	for addr := range s.addrStore {
		addrsToCheck = append(addrsToCheck, addr)
	}
	s.lock.RUnlock()

	// 检查每个地址的健康状态
	for _, addr := range addrsToCheck {
		if !s.isServiceHealthy(addr) {
			slog.Warn("检测到不健康的服务", "地址", addr)
			s.handleUnhealthyService(addr)
		}
	}
}

// isServiceHealthy 检查指定地址的服务是否健康
func (s *Service) isServiceHealthy(addr string) bool {
	s.lock.RLock()
	client := s.addrStore[addr]
	s.lock.RUnlock()

	if client == nil {
		slog.Error("RPC客户端不存在", "地址", addr)
		return false
	}

	_, err := client.Ping(context.Background(), &PingReq{})
	if err != nil {
		slog.Error("服务不健康", "错误", err, "地址", addr)
		return false
	}
	return true
}

// handleUnhealthyService 处理不健康的服务
// 清理相关的映射关系并通知订阅者
func (s *Service) handleUnhealthyService(addr string) {
	slog.Info("开始处理不健康服务", "地址", addr)

	s.lock.Lock()

	// 获取该地址提供的服务列表
	serviceNames := s.sAddrToSNames[addr]

	// 关闭并移除RPC客户端
	if client := s.addrStore[addr]; client != nil {
		//client.Close()
		delete(s.addrStore, addr)
	}

	// 从服务名到地址的映射中移除该地址
	for _, serviceName := range serviceNames {
		addrs := s.sNameToSAddr[serviceName]
		for i, serviceAddr := range addrs {
			if serviceAddr == addr {
				// 移除该地址
				s.sNameToSAddr[serviceName] = append(addrs[:i], addrs[i+1:]...)
				break
			}
		}

		// 如果服务没有任何可用实例了，清理整个条目
		if len(s.sNameToSAddr[serviceName]) == 0 {
			delete(s.sNameToSAddr, serviceName)
		}
	}

	// 从地址到服务名的映射中移除该地址
	delete(s.sAddrToSNames, addr)

	s.lock.Unlock()

	// 通知所有受影响服务的订阅者
	for _, serviceName := range serviceNames {
		slog.Info("通知服务变更", "服务名", serviceName, "移除地址", addr)
		s.notifySubscribers(&NotifyReq{
			Type:   "del",
			Server: serviceName,
			Addr:   addr,
		})
	}

	slog.Info("不健康服务处理完成", "地址", addr, "影响的服务", serviceNames)
}
