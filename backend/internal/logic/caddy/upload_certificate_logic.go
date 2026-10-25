package caddy

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

// certificateUploadFilesCtxKey multipart 文件内容的 ctx 键，
// handler 解析后传入 logic，避免 logic 再触碰 *http.Request。
type certificateUploadFilesCtxKey struct{}

// CertificateUploadFiles 一次上传的证书与私钥 PEM 内容。
type CertificateUploadFiles struct {
	CertPEM []byte
	KeyPEM  []byte
}

// WithCertificateUploadFiles 把上传文件内容挂进 ctx。
func WithCertificateUploadFiles(ctx context.Context, files *CertificateUploadFiles) context.Context {
	return context.WithValue(ctx, certificateUploadFilesCtxKey{}, files)
}

func certificateUploadFilesFrom(ctx context.Context) *CertificateUploadFiles {
	if ctx == nil {
		return nil
	}
	files, _ := ctx.Value(certificateUploadFilesCtxKey{}).(*CertificateUploadFiles)
	return files
}

type UploadCertificateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadCertificateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadCertificateLogic {
	return &UploadCertificateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UploadCertificate 校验证书与私钥配对后落盘并入库（source=upload）。
// 落盘路径 /config/certs/<serverID>/<主域名>/{fullchain.pem,privkey.pem}（0600），
// 私钥内容不回显、不进日志；仅本机服务器可上传。
func (l *UploadCertificateLogic) UploadCertificate(req *types.CertificateUploadReq) (resp *types.BaseResp, err error) {
	files := certificateUploadFilesFrom(l.ctx)
	if files == nil || len(files.CertPEM) == 0 || len(files.KeyPEM) == 0 {
		return nil, xerr.NewBusinessErrorWith("请同时上传证书与私钥 PEM 文件")
	}
	if l.svcCtx.CertificateModel == nil {
		return nil, xerr.NewSystemErrorWith("证书模型未初始化")
	}

	serverID := uint(0)
	if req != nil {
		serverID = req.ServerId
	}
	server, err := resolveLocalCertServer(l.svcCtx.DB.WithContext(l.ctx), serverID)
	if err != nil {
		return nil, err
	}

	// 配对校验：证书与私钥不匹配直接拒绝，不落任何文件
	if _, pairErr := tls.X509KeyPair(files.CertPEM, files.KeyPEM); pairErr != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("证书与私钥不匹配: %v", pairErr))
	}
	info, err := parseCertificatePEM(files.CertPEM)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(fmt.Sprintf("证书文件无法解析: %v", err))
	}

	domains := info.AllDomains()
	primaryDomain := info.PrimaryDomain()
	// 用户声明的域名在证书 SAN 覆盖范围内时，允许指定主域名展示顺序
	declaredRaw := ""
	if req != nil {
		declaredRaw = req.Domains
	}
	declared := parseUploadDeclaredDomains(declaredRaw)
	if len(declared) > 0 {
		covered := make(map[string]struct{}, len(domains))
		for _, domain := range domains {
			covered[domain] = struct{}{}
		}
		for _, domain := range declared {
			if _, ok := covered[domain]; ok {
				primaryDomain = domain
				break
			}
		}
		domains = mergeCertDomainList(domains, declared)
	}

	certPath := filepath.Join(certUploadDir(server.ID, primaryDomain), certUploadCertFileName)
	keyPath := filepath.Join(certUploadDir(server.ID, primaryDomain), certUploadKeyFileName)
	if err := writeCertPair(certPath, keyPath, files.CertPEM, files.KeyPEM); err != nil {
		return nil, xerr.NewSystemErrorWith(err.Error())
	}

	now := time.Now()
	row := &caddymodel.Certificate{
		ServerID:      server.ID,
		PrimaryDomain: primaryDomain,
		Domains:       strings.Join(domains, ","),
		Issuer:        info.Issuer,
		Source:        caddymodel.CertSourceUpload,
		NotBefore:     info.NotBefore,
		NotAfter:      info.NotAfter,
		StoragePath:   certPath,
		LastSyncedAt:  now,
	}
	row.Status, _ = certStatusAndDaysLeft(now, info.NotAfter)
	if err := l.svcCtx.CertificateModel.UpsertBatch(l.ctx, []*caddymodel.Certificate{row}); err != nil {
		// 落库失败须清掉已写文件，不留无库存记录的孤儿证书
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
		return nil, xerr.NewSystemErrorWith("写入证书库存失败")
	}

	l.Logger.Infof("[证书中心] 上传证书成功: server=%s domain=%s", server.Name, primaryDomain)
	return &types.BaseResp{Code: 0, Msg: "上传成功"}, nil
}

// parseUploadDeclaredDomains 拆出用户声明的补充域名（逗号/空白分隔）。
func parseUploadDeclaredDomains(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == ';'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = caddymodel.NormalizeCertDomain(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// mergeCertDomainList 合并证书 SAN 与用户声明域名，去重且保持先来顺序。
func mergeCertDomainList(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))
	for _, group := range [][]string{base, extra} {
		for _, domain := range group {
			if _, ok := seen[domain]; ok {
				continue
			}
			seen[domain] = struct{}{}
			result = append(result, domain)
		}
	}
	return result
}

// writeCertPair 写出证书与私钥，目录 0700、文件 0600；私钥仅本进程与 Caddy 可读。
func writeCertPair(certPath, keyPath string, certPEM, keyPEM []byte) error {
	certDir := filepath.Dir(certPath)
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("写入证书文件失败: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("写入私钥文件失败: %w", err)
	}
	return nil
}
