package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// OpenObserveHandler 是一个将日志发送到OpenObserve的slog.Handler实现
type OpenObserveHandler struct {
	opts           OpenObserveOptions //连接信息
	httpClient     *http.Client       //http客户端
	addSource      bool
	addSourceLevel slog.Level //添加源码位置的日志等级

	// 批量处理相关字段
	mu          sync.Mutex
	buffer      []map[string]interface{}
	flushTicker *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	handler slog.Handler
}

// OpenObserveOptions 配置OpenObserve连接选项
type OpenObserveOptions struct {
	// OpenObserve服务地址
	Endpoint string
	// 组织名称
	Organization string
	// 流/索引名称
	Stream string
	// 用户名
	Username string
	// 密码
	Password string
	// 请求超时时间
	Timeout time.Duration
	// 日志发送间隔，每隔该时间发送一次日志
	FlushInterval time.Duration

	Handler slog.Handler
}

// Handle 处理日志记录事件
func (h *OpenObserveHandler) Handle(ctx context.Context, record slog.Record) error {
	level := record.Level
	if !h.Enabled(ctx, level) {
		return nil
	}

	// 构建日志对象
	logEntry := make(map[string]interface{})

	// 添加基本字段
	logEntry["level"] = level.String()
	logEntry["message"] = record.Message
	// 添加源代码位置信息（如果启用）
	if level.Level() >= h.addSourceLevel && record.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			logEntry["source"] = map[string]interface{}{
				"function": f.Function,
				"file":     f.File,
				"line":     f.Line,
			}
		}
	}

	//value := ctx.Value("TraceID")
	//record.AddAttrs(slog.Any("traceID", value))
	// 上下文属性
	recordAddAttrs(ctx, &record)

	// 添加组属性
	record.Attrs(func(attr slog.Attr) bool {
		key := attr.Key
		logEntry[key] = attr.Value.Any()
		return true
	})

	// 将日志加入缓冲区
	h.mu.Lock()
	h.buffer = append(h.buffer, logEntry)
	h.mu.Unlock()

	return h.handler.Handle(ctx, record)
}

// NewOpenObserveHandler 创建一个新的OpenObserve日志处理器
func NewOpenObserveHandler(opts OpenObserveOptions, addSourceLevel slog.Level) *OpenObserveHandler {
	// 使用默认值填充未设置的选项
	if opts.Endpoint == "" {
		panic("OpenObserveOptions.Endpoint is required")
	}
	if opts.Organization == "" {
		panic("OpenObserveOptions.Organization is required")
	}
	if opts.Stream == "" {
		panic("OpenObserveOptions.Stream is required")
	}
	if opts.Username == "" {
		panic("OpenObserveOptions.Username is required")
	}
	if opts.Password == "" {
		panic("OpenObserveOptions.Password is required")
	}
	if opts.Handler == nil {
		panic("OpenObserveOptions.Handler is required")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.FlushInterval == 0 {
		opts.FlushInterval = 2 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	h := &OpenObserveHandler{
		opts:           opts,
		httpClient:     &http.Client{Timeout: opts.Timeout},
		addSourceLevel: addSourceLevel,
		buffer:         make([]map[string]interface{}, 0, 100), // 使用固定的初始容量
		ctx:            ctx,
		cancel:         cancel,
		handler:        opts.Handler,
	}
	// 启动定时发送协程
	h.flushTicker = time.NewTicker(opts.FlushInterval)
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.flushLoop()
	}()

	return h
}

// flushLoop 定时刷新日志到OpenObserve
func (h *OpenObserveHandler) flushLoop() {
	for {
		select {
		case <-h.flushTicker.C:
			go h.Flush() // 异步发送日志
		case <-h.ctx.Done():
			h.Flush() // 退出前确保所有日志都发送出去
			return
		}
	}
}

// Close 关闭处理器，发送所有缓存的日志并释放资源
func (h *OpenObserveHandler) Close() error {
	h.cancel()
	h.flushTicker.Stop()
	h.wg.Wait()
	return nil
}

// Flush 立即发送所有缓存的日志
func (h *OpenObserveHandler) Flush() {
	h.mu.Lock()

	if len(h.buffer) == 0 {
		//println("len(h.buffer) == 0")
		h.mu.Unlock()
		return
	}

	// 取出当前缓冲区中的所有日志
	logs := h.buffer
	h.buffer = make([]map[string]interface{}, 0, 100) // 重置缓冲区
	h.mu.Unlock()

	// 发送日志批次
	err := h.sendBatch(context.Background(), logs)
	if err != nil {
		// 如果发送失败，可以考虑重试或记录错误
		fmt.Printf("Failed to send logs to OpenObserve: %v\n", err)
	}
}

// Enabled 判断指定级别的日志是否应被记录
func (h *OpenObserveHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.handler.Enabled(context.Background(), level)
}

// sendBatch 批量发送日志到OpenObserve
func (h *OpenObserveHandler) sendBatch(ctx context.Context, logEntries []map[string]interface{}) error {
	if len(logEntries) == 0 {
		//println("len(logEntries) == 0")
		return nil
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/api/%s/%s/_json", h.opts.Endpoint, h.opts.Organization, h.opts.Stream)
	//fmt.Println("url:", url)
	// 将日志条目序列化为NDJSON格式
	marshal, _ := json.Marshal(logEntries)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(marshal))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.SetBasicAuth(h.opts.Username, h.opts.Password)
	// 发送请求
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending logs to OpenObserve: %w", err)
	}
	defer resp.Body.Close()
	// 检查响应状态
	if resp.StatusCode >= 400 {
		return fmt.Errorf("error from OpenObserve API: status code %d", resp.StatusCode)
	}

	return nil
}

// WithAttrs 返回一个新的添加了指定属性的Handler
func (h *OpenObserveHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	panic("not implemented")
	//if len(attrs) == 0 {
	//	return h
	//}
	//h2 := *h
	//h2.attrs = append(h.attrs[:], attrs...)
	//h2.flushLoop()
	//return &h2
}

// WithGroup 返回一个新的添加了指定组的Handler
func (h *OpenObserveHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
	//if name == "" {
	//	return h
	//}
	//h2 := *h
	//h2.groups = append(h.groups[:], name)
	//return &h2
}
