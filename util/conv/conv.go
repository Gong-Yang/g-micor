package conv

import "context"

// 延迟转换器  嵌套循环转换数据时 可以通过延迟转换器来分割阶段  增加代码可读性
type delayConverter[T any, R any] struct {
	ctx      context.Context
	convTask []*convOne[T, R]
}

type convOne[T any, R any] struct {
	form     []T
	receiver []R
}

func NewDelayConverter[T any, R any](ctx context.Context) *delayConverter[T, R] {
	return &delayConverter[T, R]{
		convTask: make([]*convOne[T, R], 0),
		ctx:      ctx,
	}
}
func (c *delayConverter[T, R]) StoreTask(form []T, receiver []R) {
	c.convTask = append(c.convTask, &convOne[T, R]{
		form:     form,
		receiver: receiver,
	})
}

func (c *delayConverter[T, R]) Conv(fn func(ctx context.Context, form T) (R, error)) error {
	for _, task := range c.convTask {
		for _, f := range task.form {
			r, err := fn(c.ctx, f)
			if err != nil {
				return err
			}
			task.receiver = append(task.receiver, r)
		}
	}
	return nil
}
