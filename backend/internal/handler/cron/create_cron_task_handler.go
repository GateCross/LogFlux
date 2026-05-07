package cron

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	cronutil "logflux/common/cron"
	"logflux/common/result"
	"logflux/internal/logic/cron"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreateCronTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if strings.Contains(contentType, "multipart/form-data") {
			r.Body = http.MaxBytesReader(w, r.Body, maxCronTaskScriptUploadRequestBytes())
			req, uploadCtx, err := parseCronTaskCreateMultipart(ctx, r, svcCtx)
			if err != nil {
				httpx.ErrorCtx(ctx, w, err)
				return
			}

			l := cron.NewCreateCronTaskLogic(uploadCtx, svcCtx)
			resp, err := l.CreateCronTask(req)
			if err != nil {
				cleanupCronTaskStagedUpload(uploadCtx)
			}
			result.HttpResult(r, w, resp, err)
			return
		}

		var req types.CronTaskReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cron.NewCreateCronTaskLogic(r.Context(), svcCtx)
		resp, err := l.CreateCronTask(&req)
		result.HttpResult(r, w, resp, err)
	}
}

func parseCronTaskCreateMultipart(ctx context.Context, r *http.Request, svcCtx *svc.ServiceContext) (*types.CronTaskReq, context.Context, error) {
	if err := r.ParseMultipartForm(maxCronTaskScriptUploadRequestBytes()); err != nil {
		return nil, ctx, fmt.Errorf("解析 multipart 表单失败: %w", err)
	}

	status, err := parseCronTaskIntFormValue(r, "status", 1)
	if err != nil {
		return nil, ctx, err
	}
	timeout, err := parseCronTaskIntFormValue(r, "timeout", 60)
	if err != nil {
		return nil, ctx, err
	}

	scriptMode := strings.TrimSpace(r.FormValue("scriptMode"))
	if scriptMode == "" {
		scriptMode = cronutil.ScriptModeFile
	}
	req := &types.CronTaskReq{
		Name:       strings.TrimSpace(r.FormValue("name")),
		Schedule:   strings.TrimSpace(r.FormValue("schedule")),
		ScriptMode: scriptMode,
		Script:     strings.TrimSpace(r.FormValue("script")),
		Status:     status,
		Timeout:    timeout,
	}

	uploadCtx, err := storeCronTaskScriptUpload(ctx, r, svcCtx)
	if err != nil {
		return nil, ctx, err
	}
	return req, uploadCtx, nil
}

func parseCronTaskIntFormValue(r *http.Request, key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s 参数必须是数字", key)
	}
	return value, nil
}

func cleanupCronTaskStagedUpload(ctx context.Context) {
	tempPath := cronutil.UploadTempPathFromContext(ctx)
	if tempPath != "" {
		_ = os.Remove(tempPath)
	}
}
