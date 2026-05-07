package cron

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"
	cronutil "logflux/common/cron"
	"logflux/common/result"
	"logflux/internal/logic/cron"
	"logflux/internal/svc"
	"logflux/internal/types"
)

const (
	cronTaskScriptMaxFileBytes         = 1 << 20
	cronTaskScriptMaxRequestExtraBytes = 1 << 20
)

func UploadCronTaskScriptHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if !strings.Contains(contentType, "multipart/form-data") {
			httpx.ErrorCtx(ctx, w, fmt.Errorf("请使用 multipart/form-data 上传脚本文件"))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxCronTaskScriptUploadRequestBytes())
		req, uploadCtx, err := parseCronTaskScriptMultipart(ctx, r, svcCtx)
		if err != nil {
			httpx.ErrorCtx(ctx, w, err)
			return
		}

		l := cron.NewUploadCronTaskScriptLogic(uploadCtx, svcCtx)
		resp, err := l.UploadCronTaskScript(req)
		result.HttpResult(r, w, resp, err)
	}
}

func parseCronTaskScriptMultipart(ctx context.Context, r *http.Request, svcCtx *svc.ServiceContext) (*types.CronTaskScriptUploadReq, context.Context, error) {
	requestBytes := maxCronTaskScriptUploadRequestBytes()
	if err := r.ParseMultipartForm(requestBytes); err != nil {
		return nil, ctx, fmt.Errorf("解析 multipart 表单失败: %w", err)
	}

	var req types.CronTaskScriptUploadReq
	if err := httpx.Parse(r, &req); err != nil {
		return nil, ctx, err
	}

	uploadCtx, err := storeCronTaskScriptUpload(ctx, r, svcCtx)
	if err != nil {
		return nil, ctx, err
	}
	return &req, uploadCtx, nil
}

func storeCronTaskScriptUpload(ctx context.Context, r *http.Request, svcCtx *svc.ServiceContext) (context.Context, error) {
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return ctx, fmt.Errorf("上传文件不能为空")
	}
	defer file.Close()

	baseDir := cronutil.FilesBaseDir(&svcCtx.Config)
	if err := cronutil.EnsureWorkspace(baseDir); err != nil {
		return ctx, fmt.Errorf("准备脚本目录失败: %w", err)
	}

	tempName := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), cronutil.SafeFileName(fileHeader.Filename))
	tempPath := filepath.Join(cronutil.TempDir(baseDir), tempName)
	targetFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return ctx, fmt.Errorf("创建临时上传文件失败: %w", err)
	}

	limitedFile := &io.LimitedReader{R: file, N: cronTaskScriptMaxFileBytes + 1}
	writtenBytes, err := io.Copy(targetFile, limitedFile)
	if err != nil {
		_ = targetFile.Close()
		_ = os.Remove(tempPath)
		return ctx, fmt.Errorf("保存上传文件失败: %w", err)
	}
	if writtenBytes > cronTaskScriptMaxFileBytes {
		_ = targetFile.Close()
		_ = os.Remove(tempPath)
		return ctx, fmt.Errorf("上传文件过大: %d > %d", writtenBytes, cronTaskScriptMaxFileBytes)
	}
	if err := targetFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return ctx, fmt.Errorf("关闭上传文件失败: %w", err)
	}

	uploadCtx := cronutil.WithUploadTempPath(ctx, tempPath)
	uploadCtx = cronutil.WithUploadFileName(uploadCtx, fileHeader.Filename)
	return uploadCtx, nil
}

func maxCronTaskScriptUploadRequestBytes() int64 {
	return cronTaskScriptMaxFileBytes + cronTaskScriptMaxRequestExtraBytes
}
