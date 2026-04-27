package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"logflux/common/logging"
	"logflux/internal/config"
	"logflux/internal/handler"
	caddylogic "logflux/internal/logic/caddy"
	"logflux/internal/middleware"
	"logflux/internal/response"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/xerr"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/config.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		httpx.OkJsonCtx(r.Context(), w, response.Error(xerr.Unauthorized, xerr.MapErrMsg(xerr.Unauthorized)))
	}))
	defer server.Stop()

	// 使用自定义控制台日志格式，统一输出风格
	plainWriter := logging.NewPlainConsoleWriter(nil, c.Log.TimeFormat)
	if strings.EqualFold(c.Log.Mode, "file") || strings.EqualFold(c.Log.Mode, "volume") {
		logx.AddWriter(plainWriter)
	} else {
		logx.SetWriter(plainWriter)
	}

	ctx := svc.NewServiceContext(c)
	if ctx.WafScheduler != nil {
		ctx.WafScheduler.SetExecutor(&wafScheduleExecutor{svcCtx: ctx})
		ctx.WafScheduler.Start()
		defer ctx.WafScheduler.Stop()
	}
	handler.RegisterHandlers(server, ctx)

	// Global Response Middleware
	server.Use(middleware.ResponseMiddleware)

	// Global Error Handler (still needed for business errors)

	httpx.SetErrorHandler(func(err error) (int, any) {
		return http.StatusOK, response.ErrorFromErr(err)
	})

	logx.Infof("Starting server at %s:%d...", c.Host, c.Port)
	server.Start()
}

type wafScheduleExecutor struct {
	svcCtx *svc.ServiceContext
}

func (executor *wafScheduleExecutor) CheckSource(ctx context.Context, sourceID uint) error {
	if executor == nil || executor.svcCtx == nil {
		return fmt.Errorf("waf scheduler svc context is nil")
	}
	ctx = caddylogic.WithWafJobTriggerMode(ctx, "schedule")
	logic := caddylogic.NewCheckWafSourceLogic(ctx, executor.svcCtx)
	_, err := logic.CheckWafSource(&types.WafSourceActionReq{ID: sourceID})
	return err
}

func (executor *wafScheduleExecutor) SyncSource(ctx context.Context, sourceID uint, activateNow bool) error {
	if executor == nil || executor.svcCtx == nil {
		return fmt.Errorf("waf scheduler svc context is nil")
	}
	ctx = caddylogic.WithWafJobTriggerMode(ctx, "schedule")
	logic := caddylogic.NewSyncWafSourceLogic(ctx, executor.svcCtx)
	_, err := logic.SyncWafSource(&types.WafSourceSyncReq{
		ID:          sourceID,
		ActivateNow: activateNow,
	})
	return err
}
