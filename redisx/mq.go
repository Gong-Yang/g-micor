package redisx

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

var ConsumerName string

func NewMq[T proto.Message](stream string, maxLen int64) *Mq[T] {
	m := &Mq[T]{
		Stream: stream,
		MaxLen: maxLen,
		wg:     &sync.WaitGroup{},
		stopCh: make(chan struct{}),
		groups: make(map[string]bool),
	}
	m.wg.Add(1)
	initList = append(initList, m)
	return m
}

type Mq[T proto.Message] struct {
	Stream string
	MaxLen int64
	wg     *sync.WaitGroup
	stopCh chan struct{} // 用于优雅关闭
	groups map[string]bool
}

func (m *Mq[T]) init() {
	m.wg.Done()
}

// Stop 停止消息监听
func (m *Mq[T]) Stop() {
	close(m.stopCh)
}

// Publish  发布消息
func (m *Mq[T]) Publish(msg T) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("消息序列化失败", "stream", m.Stream, "error", err)
		return err
	}

	args := &redis.XAddArgs{
		Stream: m.Stream,
		Values: map[string]any{
			"data": data,
		},
		MaxLen: m.MaxLen,
	}

	result := Client.XAdd(context.Background(), args)
	if result.Err() != nil {
		slog.Error("发布消息失败", "stream", m.Stream, "error", result.Err())
		return result.Err()
	}

	messageId := result.Val()
	slog.Info("消息发布成功", "stream", m.Stream, "messageId", messageId)
	return nil
}

// Listen 监听消息,并且执行方法
func (m *Mq[T]) Listen(group string, handler func(ctx context.Context, msg T) error) {
	if _, ok := m.groups[group]; ok {
		panic("消费者组已存在:" + group)
	}
	m.groups[group] = true
	go func() { //TODO go safe
		m.wg.Wait() // 等待初始化完成

		err := Client.XGroupCreateMkStream(context.Background(), m.Stream, group, "$").Err()
		//"$" 表示从最新的消息开始消费
		//消费者组只会接收到创建之后新加入 stream 的消息
		//不会处理创建之前已存在的历史消息
		//适用于只关心实时新数据的场景

		//"0" 表示从第一条消息开始消费
		//消费者组会接收到 stream 中所有历史消息（从第一条开始）
		//包括消费者组创建之前就已经存在的消息
		//适用于需要处理完整数据历史的场景

		if err != nil {
			if err.Error() == "BUSYGROUP Consumer Group name already exists" {
				slog.Info("消费者组已存在", "stream", m.Stream, "group", group)
			} else {
				panic(err)
			}
		}
		slog.Info("消费者组初始化成功", "stream", m.Stream, "group", group)

		ctx := context.Background()
		for {
			// 检查是否需要停止
			select {
			case <-m.stopCh:
				slog.Info("消息监听停止", "stream", m.Stream)
				return
			default:
			}

			result := Client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: ConsumerName,
				Streams:  []string{m.Stream, ">"},
				Count:    1,
				Block:    time.Second * 20,
			})

			if err := result.Err(); err != nil {
				// 超时无消息时返回 redis.Nil，继续轮询
				if errors.Is(err, redis.Nil) {
					continue
				}
				slog.Error("读取消息失败", "stream", m.Stream, "group", group, "error", err)
				continue
			}

			streams := result.Val()
			for _, s := range streams {
				for _, msg := range s.Messages {
					//将group放入上下文
					msgctx := context.WithValue(ctx, "TraceID", msg.ID)

					slog.InfoContext(msgctx, "MQ收到消息", "stream", m.Stream, "group", group)
					// 将消息内容转换为泛型 T
					entity := getEntity[T]()
					str, ok := msg.Values["data"].(string)
					if !ok {
						slog.ErrorContext(msgctx, "MQ消息转换失败", "stream", m.Stream, "group", group, "error", "data is not bytes")
						continue
					}
					bytes := []byte(str)
					err := proto.Unmarshal(bytes, entity)
					//converted, err := jsonx.Convert[T](msg.Values)
					if err != nil {
						slog.ErrorContext(msgctx, "消息转换失败", "stream", m.Stream, "group", group, "error", err, "values", msg.Values)
						// 转换失败不进行 ack，保留在 pending
						continue
					}

					// 执行用户处理逻辑
					if err := handler(msgctx, entity); err != nil {
						slog.ErrorContext(msgctx, "消息处理失败", "stream", m.Stream, "group", group, "error", err)
						// 处理失败不进行 ack，保留在 pending
						continue
					}

					// 处理成功后 ack
					ack := Client.XAck(ctx, m.Stream, group, msg.ID)
					if ack.Err() != nil {
						slog.ErrorContext(msgctx, "消息ACK失败", "stream", m.Stream, "group", group, "error", ack.Err())
						continue
					}
					slog.InfoContext(msgctx, "消息ACK成功", "stream", m.Stream, "group", group)
				}
			}
		}
	}()
}
