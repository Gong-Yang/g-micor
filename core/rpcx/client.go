package rpcx

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/rpc"
	"sync"
)

type Client struct {
	*sync.RWMutex
	clients []*rpc.Client
}

// NewClient 连接全部地址 生成一个client
func NewClient(addr []string) (*Client, error) {
	slog.Info("Creating new RPC client", "addresses", addr, "count", len(addr))

	if len(addr) == 0 {
		slog.Error("Failed to create RPC client: empty address list")
		return nil, errors.New("address list cannot be empty")
	}

	client := &Client{
		RWMutex: &sync.RWMutex{},
		clients: make([]*rpc.Client, 0, len(addr)),
	}

	// 尝试连接所有地址
	for _, address := range addr {
		client.Dial(address)
	}

	// 检查是否至少有一个连接成功
	client.RLock()
	hasConnection := len(client.clients) > 0
	client.RUnlock()

	if !hasConnection {
		slog.Error("Failed to connect to any server", "addresses", addr, "total_addresses", len(addr))
		return nil, fmt.Errorf("failed to connect to any of the provided addresses: %v", addr)
	}

	return client, nil
}

// Dial 添加一个地址
func (c *Client) Dial(addr string) {
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		slog.Warn("Failed to connect to server", "address", addr, "error", err)
		// 连接失败时记录错误但不阻止程序运行
		return
	}

	c.Lock()
	defer c.Unlock()
	c.clients = append(c.clients, client)

	slog.Info("Successfully connected to server",
		"address", addr)
	return
}

// Call 负载均衡调用，调用时发现连接有问题则清理该连接，使用其他连接调用 ， clients数组为空时返回err
func (c *Client) Call(serviceMethod string, args any, reply any) error {
	c.RLock()
	if len(c.clients) == 0 {
		c.RUnlock()
		slog.Error("RPC call failed: no available connections", "method", serviceMethod)
		return errors.New("no available connections")
	}
	c.RUnlock()

	for len(c.clients) > 0 {
		c.RLock()
		if len(c.clients) == 0 {
			c.RUnlock()
			slog.Error("RPC call failed: all connections lost during retries",
				"method", serviceMethod)
			return errors.New("no available connections")
		}
		// 随机选择一个客户端
		index := rand.Intn(len(c.clients))
		client := c.clients[index]
		c.RUnlock()

		// 尝试调用
		err := client.Call(serviceMethod, args, reply)
		if err == nil {
			return nil // 调用成功
		}

		slog.Warn("RPC call failed, removing connection",
			"method", serviceMethod,
			"error", err)

		// 调用失败，移除有问题的连接
		c.removeClient(index)
	}

	slog.Error("RPC call failed after all retries",
		"method", serviceMethod)
	return errors.New("all connection attempts failed")
}

// removeClient 移除指定索引的客户端连接
func (c *Client) removeClient(index int) {
	c.Lock()
	defer c.Unlock()

	if index >= 0 && index < len(c.clients) {
		// 关闭连接
		c.clients[index].Close()
		// 从切片中移除
		c.clients = append(c.clients[:index], c.clients[index+1:]...)
	}
}

// Close 关闭所有连接
func (c *Client) Close() error {
	c.Lock()
	defer c.Unlock()

	for _, client := range c.clients {
		client.Close()
	}
	c.clients = c.clients[:0]
	return nil
}

// GetConnectionCount 获取当前可用连接数
func (c *Client) GetConnectionCount() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.clients)
}
