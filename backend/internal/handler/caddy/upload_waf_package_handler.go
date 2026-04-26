package caddy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"logflux/common/result"
	logiccaddy "logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/waf"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadWafPackageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.WafUploadReq

		ctx := r.Context()
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if strings.Contains(contentType, "multipart/form-data") {
			r.Body = http.MaxBytesReader(w, r.Body, maxWafUploadRequestBytes(svcCtx))
			parsedReq, uploadCtx, err := parseWafUploadMultipart(ctx, r, svcCtx)
			if err != nil {
				httpx.ErrorCtx(ctx, w, err)
				return
			}
			req = *parsedReq
			ctx = uploadCtx
		} else {
			if err := httpx.Parse(r, &req); err != nil {
				httpx.ErrorCtx(ctx, w, err)
				return
			}
		}

		l := logiccaddy.NewUploadWafPackageLogic(ctx, svcCtx)
		resp, err := l.UploadWafPackage(&req)
		result.HttpResult(r, w, resp, err)
	}
}

func parseWafUploadMultipart(ctx context.Context, r *http.Request, svcCtx *svc.ServiceContext) (*types.WafUploadReq, context.Context, error) {
	maxBytes := svcCtx.Config.Waf.MaxPackageBytes
	if maxBytes <= 0 {
		maxBytes = waf.DefaultMaxPackageBytes
	}

	if err := r.ParseMultipartForm(maxBytes); err != nil {
		return nil, ctx, fmt.Errorf("parse multipart form failed: %w", err)
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return nil, ctx, fmt.Errorf("upload file is required")
	}
	defer file.Close()

	store := waf.NewStore(svcCtx.Config.Waf.WorkDir)
	if err := store.EnsureDirs(); err != nil {
		return nil, ctx, fmt.Errorf("prepare upload workspace failed: %w", err)
	}

	tempName := fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), filepathSafeBase(fileHeader.Filename))
	tempPath := store.StagePath(tempName)
	targetFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, ctx, fmt.Errorf("create temp upload file failed: %w", err)
	}
	limitedFile := &io.LimitedReader{R: file, N: maxBytes + 1}
	writtenBytes, err := io.Copy(targetFile, limitedFile)
	if err != nil {
		_ = targetFile.Close()
		_ = os.Remove(tempPath)
		return nil, ctx, fmt.Errorf("save upload file failed: %w", err)
	}
	if writtenBytes > maxBytes {
		_ = targetFile.Close()
		_ = os.Remove(tempPath)
		return nil, ctx, fmt.Errorf("upload package too large: %d > %d", writtenBytes, maxBytes)
	}
	if err := targetFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, ctx, fmt.Errorf("close upload file failed: %w", err)
	}

	activateNow := false
	if rawValue := strings.TrimSpace(r.FormValue("activateNow")); rawValue != "" {
		parsed, parseErr := strconv.ParseBool(rawValue)
		if parseErr != nil {
			_ = os.Remove(tempPath)
			return nil, ctx, fmt.Errorf("invalid activateNow value")
		}
		activateNow = parsed
	}

	req := &types.WafUploadReq{
		Kind:        strings.TrimSpace(r.FormValue("kind")),
		Version:     strings.TrimSpace(r.FormValue("version")),
		Checksum:    strings.TrimSpace(r.FormValue("checksum")),
		ActivateNow: activateNow,
	}
	if req.Kind == "" {
		req.Kind = "crs"
	}

	uploadCtx := context.WithValue(ctx, "waf_upload_temp_path", tempPath)
	uploadCtx = context.WithValue(uploadCtx, "waf_upload_file_name", fileHeader.Filename)
	return req, uploadCtx, nil
}

func maxWafUploadRequestBytes(svcCtx *svc.ServiceContext) int64 {
	maxBytes := svcCtx.Config.Waf.MaxPackageBytes
	if maxBytes <= 0 {
		maxBytes = waf.DefaultMaxPackageBytes
	}
	// multipart 字段和边界会带来少量额外开销，给表单元数据预留 1MiB。
	return maxBytes + 1024*1024
}

func filepathSafeBase(name string) string {
	base := strings.TrimSpace(name)
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, "..", "_")
	if base == "" {
		return "package"
	}
	return base
}
