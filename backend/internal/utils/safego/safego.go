package safego

import (
	"context"
	"runtime/debug"

	"github.com/zeromicro/go-zero/core/logx"
)

// Task 用于启动带 recover 的后台 goroutine。
type Task struct {
	ctx  context.Context
	name string
}

// New 创建安全 goroutine 任务。
func New(ctx context.Context, name string) *Task {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Task{ctx: ctx, name: name}
}

// Go 启动 goroutine，并在 panic 时记录中文错误日志。
func (t *Task) Go(fn func()) {
	if t == nil {
		t = New(context.Background(), "未命名任务")
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logx.WithContext(t.ctx).Errorf("后台任务发生 panic: task=%s err=%v stack=%s", t.name, recovered, string(debug.Stack()))
			}
		}()
		fn()
	}()
}
