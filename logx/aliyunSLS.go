package logx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
)

// AliyunSLSHandler 是一个将日志发送到阿里云 SLS (Simple Log Service) 的 slog.Handler 实现。
//
// 内部使用阿里云官方 SDK (github.com/aliyun/aliyun-log-go-sdk/producer)，
// 由它负责批量打包、失败重试、并发发送和优雅关闭，本实现只负责把
// slog.Record 翻译成 SLS 的 *sls.Log 并 SendLog 出去。
type AliyunSLSHandler struct {
	opts           AliyunSLSOptions
	addSourceLevel slog.Level

	producer *producer.Producer
	handler  slog.Handler // 本地输出，比如 slog.NewTextHandler(os.Stdout, ...)

	// 派生 handler（WithAttrs / WithGroup）共享同一个 producer 与 closed 标记
	closed *atomic.Bool

	// groups：当前所在的分组栈，最终前缀为 "a.b."
	groups []string
	// preformatted：通过 WithAttrs 累积下来的属性，key 已经按调用当时的 group 前缀拼好。
	// 为了让派生 handler 安全共享，这个 map 在派生时整体复制，运行期不再修改。
	preformatted map[string]string
}

// AliyunSLSOptions 阿里云 SLS 连接与行为配置
type AliyunSLSOptions struct {
	// SLS 服务入口，例如 "cn-hangzhou.log.aliyuncs.com"
	Endpoint string
	// SLS Project 名
	Project string
	// SLS Logstore 名
	Logstore string

	// 阿里云访问凭证（静态 AK/SK）。
	// 如果使用 ECS RAM Role / STS 等动态凭证，可以直接通过
	// ProducerConfig.CredentialsProvider 注入，这两个字段留空。
	AccessKeyID     string
	AccessKeySecret string
	// 可选，使用 STS 临时凭证时填写
	SecurityToken string

	// 写入 SLS 的 topic / source，可为空
	Topic  string
	Source string

	// 本地日志 handler（同步写出），通常给一个 slog.NewTextHandler(os.Stdout, ...)
	Handler slog.Handler

	// 关闭 producer 时的最大等待毫秒数
	// > 0 时使用 producer.Close(timeoutMs)
	// <= 0 时使用 producer.SafeClose() 一直等到所有日志发送完
	CloseTimeoutMs int64

	// 可选：自定义 producer 配置，给 nil 走默认值
	// 如果设置了，仍会用 Endpoint / 凭证字段覆盖同名配置（除非已自行设置 CredentialsProvider）。
	ProducerConfig *producer.ProducerConfig
}

// NewAliyunSLSHandler 创建一个新的阿里云 SLS 日志处理器。
// addSourceLevel 表示从哪一个级别开始记录源代码位置（与 OpenObserveHandler 一致）。
func NewAliyunSLSHandler(opts AliyunSLSOptions, addSourceLevel slog.Level) *AliyunSLSHandler {
	if opts.Endpoint == "" {
		panic("AliyunSLSOptions.Endpoint is required")
	}
	if opts.Project == "" {
		panic("AliyunSLSOptions.Project is required")
	}
	if opts.Logstore == "" {
		panic("AliyunSLSOptions.Logstore is required")
	}
	if opts.Handler == nil {
		panic("AliyunSLSOptions.Handler is required")
	}

	cfg := opts.ProducerConfig
	if cfg == nil {
		cfg = producer.GetDefaultProducerConfig()
	}
	cfg.Endpoint = opts.Endpoint
	if cfg.CredentialsProvider == nil {
		if opts.AccessKeyID == "" || opts.AccessKeySecret == "" {
			panic("AliyunSLSOptions: AccessKeyID/AccessKeySecret or ProducerConfig.CredentialsProvider is required")
		}
		cfg.CredentialsProvider = sls.NewStaticCredentialsProvider(
			opts.AccessKeyID,
			opts.AccessKeySecret,
			opts.SecurityToken,
		)
	}

	p, err := producer.NewProducer(cfg)
	if err != nil {
		panic(fmt.Errorf("create aliyun sls producer failed: %w", err))
	}
	p.Start()

	return &AliyunSLSHandler{
		opts:           opts,
		addSourceLevel: addSourceLevel,
		producer:       p,
		handler:        opts.Handler,
		closed:         new(atomic.Bool),
	}
}

// Enabled 判断指定级别的日志是否应被记录，委托给本地 handler。
func (h *AliyunSLSHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录事件
func (h *AliyunSLSHandler) Handle(ctx context.Context, record slog.Record) error {
	level := record.Level
	if !h.Enabled(ctx, level) {
		return nil
	}

	// Clone 一份，避免我们后续 AddAttrs 污染调用者持有的 Record
	record = record.Clone()

	// 把 ctx 中的属性塞进 record，与 OpenObserveHandler 行为保持一致，
	// 这样本地 handler 也能输出 ctx 上的 traceID 等信息
	recordAddAttrs(ctx, &record)

	// 构建 SLS contents（key/value 都是字符串）
	contents := make(map[string]string, len(h.preformatted)+record.NumAttrs()+8)
	contents["level"] = level.String()
	contents["message"] = record.Message
	logTime := record.Time
	if logTime.IsZero() {
		logTime = time.Now()
	}
	contents["time"] = logTime.Format(time.RFC3339Nano)

	// 源代码位置
	if level >= h.addSourceLevel && record.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			contents["source.function"] = f.Function
			contents["source.file"] = f.File
			contents["source.line"] = strconv.Itoa(f.Line)
		}
	}

	// WithAttrs 累积下来的属性（key 已含拼好的 group 前缀）
	for k, v := range h.preformatted {
		contents[k] = v
	}

	// Record 自身的属性，使用当前完整的 group 前缀
	prefix := h.groupPrefix()
	record.Attrs(func(a slog.Attr) bool {
		writeAttrToContents(contents, prefix, a)
		return true
	})

	if !h.closed.Load() {
		slsLog := producer.GenerateLog(uint32(logTime.Unix()), contents)
		if err := h.producer.SendLog(
			h.opts.Project,
			h.opts.Logstore,
			h.opts.Topic,
			h.opts.Source,
			slsLog,
		); err != nil {
			// 这里不能再用 slog 记录，会出现递归
			fmt.Fprintf(os.Stderr, "aliyun sls send log failed: %v\n", err)
		}
	}

	// 同步打到本地（stdout 等）
	return h.handler.Handle(ctx, record)
}

// WithAttrs 返回一个携带额外属性的新 Handler。
// 关键点：WithAttrs 调用时立刻把 attrs 按当前 group 前缀展开为预格式化的 contents，
// 后续即便再 WithGroup，也不会影响已经定型的这批属性的 key。
func (h *AliyunSLSHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	newPreformatted := make(map[string]string, len(h.preformatted)+len(attrs))
	for k, v := range h.preformatted {
		newPreformatted[k] = v
	}
	prefix := h.groupPrefix()
	for _, a := range attrs {
		writeAttrToContents(newPreformatted, prefix, a)
	}
	return &AliyunSLSHandler{
		opts:           h.opts,
		addSourceLevel: h.addSourceLevel,
		producer:       h.producer,
		handler:        h.handler.WithAttrs(attrs),
		closed:         h.closed,
		groups:         h.groups,
		preformatted:   newPreformatted,
	}
}

// WithGroup 返回一个新加入分组的 Handler
func (h *AliyunSLSHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	// 三参切片，避免与 sibling clone 共享底层数组
	newGroups := append(h.groups[:len(h.groups):len(h.groups)], name)
	return &AliyunSLSHandler{
		opts:           h.opts,
		addSourceLevel: h.addSourceLevel,
		producer:       h.producer,
		handler:        h.handler.WithGroup(name),
		closed:         h.closed,
		groups:         newGroups,
		preformatted:   h.preformatted,
	}
}

// Close 关闭处理器，发送所有缓存的日志并释放资源。
// 多次调用是安全的，仅第一次生效。派生出来的 handler 也会随之关闭。
func (h *AliyunSLSHandler) Close() error {
	if h.closed.Swap(true) {
		return nil
	}
	if h.opts.CloseTimeoutMs > 0 {
		return h.producer.Close(h.opts.CloseTimeoutMs)
	}
	h.producer.SafeClose()
	return nil
}

// groupPrefix 把当前 group 链拼成 "a.b." 形式的前缀
func (h *AliyunSLSHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

// writeAttrToContents 把 slog.Attr 写入 SLS contents map。
// SLS 要求 contents 是 string -> string，所以要把任意类型的 value 序列化为字符串。
// 嵌套的 Group 会展开成 "group.key" 形式。
func writeAttrToContents(contents map[string]string, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	// Group：递归展开
	if a.Value.Kind() == slog.KindGroup {
		group := a.Value.Group()
		if len(group) == 0 {
			return
		}
		var p string
		if a.Key == "" {
			p = prefix
		} else {
			p = prefix + a.Key + "."
		}
		for _, ga := range group {
			writeAttrToContents(contents, p, ga)
		}
		return
	}

	contents[prefix+a.Key] = attrValueToString(a.Value)
}

// attrValueToString 把 slog.Value 转成字符串
func attrValueToString(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339Nano)
	case slog.KindAny:
		raw := v.Any()
		if raw == nil {
			return ""
		}
		if err, ok := raw.(error); ok {
			return err.Error()
		}
		if s, ok := raw.(fmt.Stringer); ok {
			return s.String()
		}
		// 其它类型用 JSON 序列化
		if b, err := json.Marshal(raw); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", raw)
	default:
		return v.String()
	}
}
