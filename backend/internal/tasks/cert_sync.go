package tasks

import (
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

// CertSyncDailySchedule 证书库存每日同步时刻（凌晨 3:40，避开归档 2 点高峰）。
const CertSyncDailySchedule = "0 40 3 * * *"

// CertSyncExecutor 证书同步执行器，由 main 注入 logic 层实现，
// 避免 tasks 直接依赖 logic 造成循环引用。
type CertSyncExecutor interface {
	SyncCertificates(ctx context.Context) error
}

// CertSyncScheduler 按固定 Cron 每日触发一次证书库存双源同步。
// 只有一个编译期固定条目，无需动态注册与移除机械。
type CertSyncScheduler struct {
	cron     *cron.Cron
	schedule string

	mu       sync.RWMutex
	executor CertSyncExecutor
	started  bool
}

func NewCertSyncScheduler() *CertSyncScheduler {
	// 与其余调度器一致：支持 5/6 位 Cron 表达式
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	return &CertSyncScheduler{
		cron:     cron.New(cron.WithParser(parser)),
		schedule: CertSyncDailySchedule,
	}
}

func (scheduler *CertSyncScheduler) SetExecutor(executor CertSyncExecutor) {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	scheduler.executor = executor
	scheduler.mu.Unlock()
}

func (scheduler *CertSyncScheduler) Start() {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	if scheduler.started {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = true
	scheduler.mu.Unlock()

	if _, err := scheduler.cron.AddFunc(scheduler.schedule, scheduler.run); err != nil {
		logx.Errorf("注册证书每日同步任务失败: schedule=%s err=%v", scheduler.schedule, err)
		return
	}

	scheduler.cron.Start()
	logx.Infof("证书同步调度器已启动，调度: %s", scheduler.schedule)
}

func (scheduler *CertSyncScheduler) Stop() {
	if scheduler == nil {
		return
	}

	scheduler.mu.Lock()
	if !scheduler.started {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = false
	scheduler.mu.Unlock()

	scheduler.cron.Stop()
	logx.Info("证书同步调度器已停止")
}

// run 单次同步失败只记日志，等待次日重试，不影响其他调度。
func (scheduler *CertSyncScheduler) run() {
	if scheduler == nil {
		return
	}

	scheduler.mu.RLock()
	executor := scheduler.executor
	scheduler.mu.RUnlock()
	if executor == nil {
		logx.Error("跳过证书每日同步，执行器为空")
		return
	}

	if err := executor.SyncCertificates(context.Background()); err != nil {
		logx.Errorf("证书每日同步失败: %v", err)
		return
	}
	logx.Info("证书每日同步完成")
}
