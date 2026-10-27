package caddy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"logflux/common/result"
	logiccaddy "logflux/internal/logic/caddy"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// certUploadMaxRequestBytes multipart 表单总大小上限：两个 PEM 文件加表单元数据。
const certUploadMaxRequestBytes = 8 << 20

// 单个 PEM 文件的读取上限，与 logic 侧约束一致。
const certUploadMaxFileBytes = 1 << 20

func UploadCertificateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CertificateUploadReq

		ctx := r.Context()
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		// 非 multipart 表单带不上证书文件，logic 侧必然失败，直接明确报错
		if !strings.Contains(contentType, "multipart/form-data") {
			httpx.ErrorCtx(ctx, w, fmt.Errorf("证书上传必须使用 multipart/form-data 表单"))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, certUploadMaxRequestBytes)
		parsedReq, uploadCtx, err := parseCertificateUploadMultipart(ctx, r)
		if err != nil {
			httpx.ErrorCtx(ctx, w, err)
			return
		}
		req = *parsedReq
		ctx = uploadCtx

		l := logiccaddy.NewUploadCertificateLogic(ctx, svcCtx)
		resp, err := l.UploadCertificate(&req)
		result.HttpResult(r, w, resp, err)
	}
}

// parseCertificateUploadMultipart 解析 cert/key 两个 PEM 文件与 domains 表单值；
// 文件内容经 ctx 传给 logic，缺失或超限直接返回中文错误。
func parseCertificateUploadMultipart(ctx context.Context, r *http.Request) (*types.CertificateUploadReq, context.Context, error) {
	if err := r.ParseMultipartForm(certUploadMaxRequestBytes); err != nil {
		return nil, ctx, fmt.Errorf("解析 multipart 表单失败: %w", err)
	}

	certPEM, err := readCertUploadFile(r, "cert", "证书")
	if err != nil {
		return nil, ctx, err
	}
	keyPEM, err := readCertUploadFile(r, "key", "私钥")
	if err != nil {
		return nil, ctx, err
	}

	req := &types.CertificateUploadReq{
		Domains: strings.TrimSpace(r.FormValue("domains")),
	}

	uploadCtx := logiccaddy.WithCertificateUploadFiles(ctx, &logiccaddy.CertificateUploadFiles{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
	})
	return req, uploadCtx, nil
}

// readCertUploadFile 读取一个上传文件字段并限制大小。
func readCertUploadFile(r *http.Request, field, label string) ([]byte, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("请上传%s PEM 文件", label)
	}
	defer file.Close()

	limited := &io.LimitedReader{R: file, N: certUploadMaxFileBytes + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("读取%s文件失败: %w", label, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s文件不能为空", label)
	}
	if len(data) > certUploadMaxFileBytes {
		return nil, fmt.Errorf("%s文件过大: 超过 %d 字节", label, certUploadMaxFileBytes)
	}
	return data, nil
}
