package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/Gong-Yang/g-micor/errorx"
	"github.com/Gong-Yang/g-micor/util/random"
	"github.com/redis/go-redis/v9"
)

var flightMap = make(map[string]complete)

type (
	complete interface {
		complete(key string)
	}
	DistributedSingleFlight[T any] struct {
		name    string
		lock    sync.Mutex
		callMap map[string]*call[T]
	}
	call[T any] struct {
		wg        *sync.WaitGroup
		one       *sync.Once
		doneChan  chan struct{}
		processId string // 执行id
		*cacheItem[T]
	}
	cacheItem[T any] struct {
		Data     T                 `json:"data,omitempty"`
		BuError  *errorx.ErrorCode `json:"buError,omitempty"`
		SysError string            `json:"sysError,omitempty"`
	}
)

func (c *call[T]) done() {
	c.one.Do(func() {
		c.wg.Done()
		close(c.doneChan)
	})
}

var isInit bool

func InitSingleFlight() {
	if isInit {
		panic("singleFlight is init")
	}
	ctx := context.Background()

	subscribe := Client.PSubscribe(ctx, "singleFlight:*")
	// 这样能保证 Init 返回后，Redis 端绝对已经准备好接收消息了
	_, err := subscribe.Receive(ctx)
	if err != nil {
		panic(fmt.Errorf("redis subscribe failed: %w", err))
	}
	channel := subscribe.Channel()
	isInit = true

	go func() {
		for {
			msg := <-channel
			flight := flightMap[msg.Channel]
			flight.complete(msg.Payload)
		}
	}()
}

func NewSingleFlight[T any](name string) *DistributedSingleFlight[T] {
	if isInit {
		panic("singleFlight is init")
	}
	name = "singleFlight:" + name
	_, ok := flightMap[name]
	if ok {
		panic(fmt.Errorf("flight %s already exists", name))
	}
	res := &DistributedSingleFlight[T]{
		name:    name,
		callMap: make(map[string]*call[T]),
	}
	flightMap[name] = res
	return res
}

func (d *DistributedSingleFlight[T]) complete(key string) {
	ctx := context.Background()
	d.lock.Lock()
	callner, ok := d.callMap[key]
	d.lock.Unlock()
	if !ok { //不需要通知
		return
	}
	defer callner.done() //通知本机协程来拿结果
	resKey := "singleFlight:res:" + callner.processId
	bytes, err := Client.Get(ctx, resKey).Bytes()
	callner.cacheItem = &cacheItem[T]{}
	if err != nil { // redis 获取失败
		slog.ErrorContext(ctx, "singleFlight get res by redis get error", "err", err)
		callner.SysError = fmt.Sprintf("singleFlight get res by redis get error %s", err.Error())
		return
	}
	err = json.Unmarshal(bytes, callner.cacheItem)
	if err != nil { // json 解析失败
		slog.ErrorContext(ctx, "singleFlight get res by redis json.Unmarshal error", "err", err)
		callner.SysError = fmt.Sprintf("singleFlight get res by redis json.Unmarshal error %s", err.Error())
		return
	}
	// 成功~
	return
}

var getSetNX = redis.NewScript(`
		local currentVal = redis.call('GET', KEYS[1])
		if currentVal then
			return {currentVal, 0}
		else
			redis.call('SET', KEYS[1], ARGV[1])
			return {ARGV[1], 1}
		end
	`)

func GetSetNX(ctx context.Context, key string, setAndReceiver any) (isSet bool, err error) {
	rv := reflect.ValueOf(setAndReceiver)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false, errors.New("setAndReceiver must be a pointer")
	}
	savedValue, err := json.Marshal(setAndReceiver)
	if err != nil {
		return false, err
	}

	// 执行脚本
	// 返回值是一个 interface{}，在 Lua 中返回 table，在 Go 中会映射为 []interface{}
	result, err := getSetNX.Run(ctx, Client, []string{key}, string(savedValue)).Result()
	if err != nil {
		panic(err)
	}

	// 解析结果
	resArray := result.([]interface{})
	val := resArray[0].(string)
	isSet = resArray[1].(int64) == 1
	if isSet {
		slog.InfoContext(ctx, "Key 不存在，已设置新值", "res", val)
	} else {
		slog.InfoContext(ctx, "Key 已存在，已获取当前值", "res", val)
	}
	err = json.Unmarshal([]byte(val), setAndReceiver)
	return isSet, err
}
func (d *DistributedSingleFlight[T]) Do(ctx context.Context, key string, fn func() (T, error)) (res T, err error) {
	d.lock.Lock()
	callner, ok := d.callMap[key]
	if ok { // 本地存在
		d.lock.Unlock()
		callner.wg.Wait()
		return callner.Result()
	}
	requestId := random.SnoyflakeString()
	isSet, err := GetSetNX(ctx, key, &requestId) //取得当前请求ID
	if err != nil {
		slog.ErrorContext(ctx, "DistributedSingleFlight GetSetNX error", "err", err)
		return
	}

	// 创建一个阻塞对象
	wg := &sync.WaitGroup{}
	wg.Add(1)
	callner = &call[T]{
		one:       &sync.Once{},
		doneChan:  make(chan struct{}),
		wg:        wg,
		processId: requestId,
	}
	d.callMap[key] = callner
	d.lock.Unlock()

	if !isSet { // 已存在其他节点在跑
		// 防止消息通知丢了，起一个协程每2秒扫一下结果
		go func() {
			ticker := time.NewTicker(time.Second * 2)
			for {
				select {
				case <-ticker.C:
					resKey := "singleFlight:res:" + callner.processId
					bytes, err := Client.Get(ctx, resKey).Bytes()
					if err != nil {
						if errors.Is(err, redis.Nil) {
							continue
						}
						// redis 有问题
						callner.SysError = err.Error()
						callner.done()
						return
					} else { // 获取结果成功
						callner.cacheItem = &cacheItem[T]{}
						err = json.Unmarshal(bytes, callner.cacheItem)
						if err != nil {
							callner.SysError = err.Error()
						}
						callner.done()
						return
					}
				case <-callner.doneChan:
					return
				}
			}
		}()
		wg.Wait()
		// 解除本地key
		d.lock.Lock()
		delete(d.callMap, key)
		d.lock.Unlock()
		return callner.Result()
	}

	// 准备跑
	// 看门狗 守住锁
	dogWg := sync.WaitGroup{}
	dogWg.Add(1)
	watchdogDone := make(chan struct{})
	//defer close(watchdogDone) //多余的
	go func() {
		defer dogWg.Done()
		tick := time.NewTicker(time.Second * 7)
		defer tick.Stop()
		for {
			select {
			case <-tick.C: //续期
				Client.Expire(ctx, key, time.Second*24)
			case <-watchdogDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	// 执行
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "PANIC in flight function", "err", r)
				pancMsg := fmt.Sprintf("singleFlight panic %s", r)
				callner.SysError = pancMsg
				callner.done()
				close(watchdogDone) // 停止看门狗
				panic(r)
			}
		}()
		res, err = fn()
	}()
	close(watchdogDone)
	dogWg.Wait() // 等待看门狗退出
	cacheObj := newCacheItem(res, err)
	callner.cacheItem = cacheObj

	callner.done() // 本机其他协程 可以先拿结果了

	// 设置远程结果
	resKey := "singleFlight:res:" + requestId
	marshal, _ := json.Marshal(cacheObj)
	Client.Set(ctx, resKey, marshal, time.Second*20)
	// 删除全局锁
	Client.Del(ctx, key)
	// 解除本地key
	d.lock.Lock()
	delete(d.callMap, key)
	d.lock.Unlock()
	// 发布通知
	Client.Publish(ctx, d.name, key)
	return
}
func (t cacheItem[T]) Result() (T, error) {
	if t.BuError != nil {
		return t.Data, *t.BuError
	}
	if t.SysError != "" {
		return t.Data, errors.New(t.SysError)
	}
	return t.Data, nil
}
func newCacheItem[T any](res T, err error) *cacheItem[T] {
	cache := &cacheItem[T]{
		Data: res,
	}
	var buErr *errorx.ErrorCode // 声明具体类型的指针
	if err != nil {
		if errors.As(err, &buErr) { // 传入指针的地址
			cache.BuError = buErr
		} else {
			cache.SysError = err.Error()
		}
	}
	return cache
}
