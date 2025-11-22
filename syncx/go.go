package syncx

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
)

// GOSafe 安全协程 内部panic不会导致整个应用挂掉
func GOSafe(ctx context.Context, f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "sub go panic", "err", r, "panic", string(debug.Stack()))
			}
		}()
		f()
	}()
}

// GoSafeWg 安全协程 自动wg.Done
func GoSafeWg(ctx context.Context, wg *sync.WaitGroup, f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "sub go panic", "err", r, "panic", string(debug.Stack()))
			}
			wg.Done()
		}()
		f()
	}()
}

type Task func()

type WorkerPool struct {
	tasks chan Task
	wg    sync.WaitGroup
	ctx   context.Context
}

// NewWorkerPool 创建一个线程池，size 是池的大小（goroutine 数量）
func NewWorkerPool(ctx context.Context, size int) *WorkerPool {
	pool := &WorkerPool{
		tasks: make(chan Task),
		ctx:   ctx,
	}
	// 启动指定数量的 worker goroutine
	for i := 0; i < size; i++ {
		GOSafe(ctx, pool.worker)
	}
	return pool
}

// worker 是每个 goroutine 的执行逻辑
func (p *WorkerPool) worker() {
	for task := range p.tasks {
		task()      // 执行任务
		p.wg.Done() // 任务完成，减少 WaitGroup 计数器
	}
}

// Submit 提交任务到线程池
func (p *WorkerPool) Submit(task Task) {
	p.wg.Add(1) // 增加 WaitGroup 计数器
	p.tasks <- task
}

// Wait 等待所有任务完成
func (p *WorkerPool) Wait() {
	p.wg.Wait() // 阻塞，直到 WaitGroup 计数器归零
}

// Close 关闭线程池
func (p *WorkerPool) Close() {
	close(p.tasks) // 关闭任务通道，停止所有 worker goroutine
}
