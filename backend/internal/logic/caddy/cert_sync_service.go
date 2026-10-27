package caddy

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	caddymodel "logflux/model/caddy"

	"logflux/internal/config"
	"logflux/internal/notification"
	"logflux/internal/svc"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// 上传证书在容器内的落盘根目录，Caddy 可直接以文件路径引用。
const certUploadBaseDir = "/config/certs"

// 上传的证书/私钥文件落盘名。
const (
	certUploadCertFileName = "fullchain.pem"
	certUploadKeyFileName  = "privkey.pem"
)

// 单个上传文件大小上限（PEM 证书远小于此值），防异常大文件拖垮内存。
const certUploadMaxFileBytes = 1 << 20

// PEM 解析的文件大小上限，跳过异常大文件避免整读浪费内存。
const certScanMaxFileBytes = 4 << 20

// CertSyncService 证书中心库存同步：卷扫描（本机真值）+ Caddyfile 声明（应有清单）双源合并。
// 面板只声明意图与展示结果，签发与续期一律由 Caddy 负责。
type CertSyncService struct {
	logger logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCertSyncService(ctx context.Context, svcCtx *svc.ServiceContext) *CertSyncService {
	return &CertSyncService{
		logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Sync 同步证书库存（单实例：只有唯一一份库存）。
// 同步即结算：卷里消失的证书下架，只有 Caddyfile 声明的行保留为 pending。
func (s *CertSyncService) Sync() error {
	if s == nil || s.svcCtx == nil {
		return xerr.NewSystemErrorWith("证书同步服务未初始化")
	}

	return s.syncInventory(s.certStorageDir())
}

func (s *CertSyncService) certStorageDir() string {
	if dir := strings.TrimSpace(s.svcCtx.Config.Caddy.CertStorageDir); dir != "" {
		return filepath.Clean(dir)
	}
	return config.DefaultCaddyCertStorageDir
}

// syncInventory 结算一次证书库存：卷扫描（真值）+ Caddyfile 声明（应有清单）。
func (s *CertSyncService) syncInventory(certDir string) error {
	currentConfig, _, _ := loadCurrentCaddyConfig(s.svcCtx.DB)
	declared, err := extractDeclaredCerts(currentConfig)
	if err != nil {
		// 声明解析失败时跳过同步并保留既有库存，待配置修复后下次同步恢复
		s.logger.Errorf("[证书中心] 解析 Caddyfile 声明失败，跳过本次同步: err=%v", err)
		return nil
	}

	// 单实例：证书目录与后端同容器同卷，恒可扫描
	scanned, err := scanLocalCertificates(certDir)
	if err != nil {
		return xerr.NewSystemErrorWith(fmt.Sprintf("扫描证书目录失败: %s err=%v", certDir, err))
	}

	model := s.svcCtx.CertificateModel
	if model == nil {
		return xerr.NewSystemErrorWith("证书模型未初始化")
	}
	existing, err := model.List(s.ctx, caddymodel.CertificateQuery{ServerID: caddymodel.LegacyServerID})
	if err != nil {
		return xerr.NewSystemErrorWith(fmt.Sprintf("查询证书库存失败: %v", err))
	}

	// 上传行以文件仍在盘上为准：消失的直接下架，存活的参与合并；
	// stat 报"文件不存在"以外的错误（卷暂缺等）保留库存待下次，不清空上传记录
	aliveUploads := make([]caddymodel.Certificate, 0, len(existing))
	deleteIDs := make([]uint, 0)
	for i := range existing {
		row := existing[i]
		if row.Source != caddymodel.CertSourceUpload {
			continue
		}
		if row.StoragePath == "" {
			deleteIDs = append(deleteIDs, row.ID)
			continue
		}
		_, statErr := os.Stat(row.StoragePath)
		switch {
		case statErr == nil:
			aliveUploads = append(aliveUploads, row)
		case os.IsNotExist(statErr):
			deleteIDs = append(deleteIDs, row.ID)
		default:
			s.logger.Errorf("[证书中心] 检查上传证书文件状态失败，保留库存待下次同步: path=%s err=%v", row.StoragePath, statErr)
			aliveUploads = append(aliveUploads, row)
		}
	}

	merged := mergeCertInventory(certMergeInput{
		ServerID: caddymodel.LegacyServerID,
		Declared: declared,
		Scanned:  scanned,
		Uploads:  aliveUploads,
		Existing: existing,
		Now:      time.Now(),
	})
	deleteIDs = append(deleteIDs, merged.DeleteIDs...)

	if len(deleteIDs) > 0 {
		if err := model.DeleteByIDs(s.ctx, deleteIDs); err != nil {
			return xerr.NewSystemErrorWith(fmt.Sprintf("下架证书失败: %v", err))
		}
		s.logger.Infof("[证书中心] 已下架证书 %d 张", len(deleteIDs))
	}
	if len(merged.Upserts) > 0 {
		if err := model.UpsertBatch(s.ctx, merged.Upserts); err != nil {
			return xerr.NewSystemErrorWith(fmt.Sprintf("写入证书库存失败: %v", err))
		}
	}
	s.notifyCertState(merged.Upserts)
	return nil
}

// notifyCertState 按剩余天数发到期通知：
// 已过期 critical、剩 1 天 critical（EventCertExpired/EventCertExpiring），
// 剩 14/7 天 warning（EventCertExpiring）；声明行（pending/missing）无真值不发。
//
// 同一张证书按「状态:级别:到期时间戳」去重（LastNotifiedKey）：每日同步与
// 手动刷新重复触发时已通知过的组合不再重发，直接匹配渠道因此绕不开的规则静默期
// 在这里补上门槛；状态跃迁（临期→过期）或证书续期（到期时间变化）换键后恢复提醒。
func (s *CertSyncService) notifyCertState(rows []*caddymodel.Certificate) {
	mgr := s.svcCtx.NotificationMgr
	if mgr == nil || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		if row.NotAfter.IsZero() {
			continue
		}
		status, daysLeft := certStatusAndDaysLeft(time.Now(), row.NotAfter)
		if status != caddymodel.CertStatusExpiring && status != caddymodel.CertStatusExpired {
			continue
		}

		var event *notification.Event
		switch {
		case status == caddymodel.CertStatusExpired:
			event = notification.NewEvent(
				notification.EventCertExpired,
				notification.LevelCritical,
				"证书已过期",
				fmt.Sprintf("证书 %s 已过期，请尽快更换", row.PrimaryDomain),
			)
		case daysLeft <= 1:
			event = notification.NewEvent(
				notification.EventCertExpiring,
				notification.LevelCritical,
				"证书即将到期",
				fmt.Sprintf("证书 %s 明天到期，请尽快更换", row.PrimaryDomain),
			)
		default:
			event = notification.NewEvent(
				notification.EventCertExpiring,
				notification.LevelWarning,
				"证书即将到期",
				fmt.Sprintf("证书 %s 距到期还有 %d 天", row.PrimaryDomain, daysLeft),
			)
		}

		notifyKey := fmt.Sprintf("%s:%s:%d", status, event.Level, row.NotAfter.Unix())
		if row.LastNotifiedKey == notifyKey {
			continue
		}
		event.WithDataMap(map[string]interface{}{
			"domain":   row.PrimaryDomain,
			"serverId": caddymodel.LegacyServerID,
			"daysLeft": daysLeft,
			"status":   status,
		})
		if err := mgr.Notify(s.ctx, event); err != nil {
			// 发送失败不落去重键，下次同步重试
			s.logger.Errorf("[证书中心] 发送证书到期通知失败: domain=%s err=%v", row.PrimaryDomain, err)
			continue
		}
		if s.svcCtx.CertificateModel != nil && row.ID > 0 {
			if err := s.svcCtx.CertificateModel.UpdateLastNotifiedKey(s.ctx, row.ID, notifyKey); err != nil {
				s.logger.Errorf("[证书中心] 记录证书通知去重键失败: domain=%s err=%v", row.PrimaryDomain, err)
			}
		}
	}
}

// ---------- 纯函数：PEM 解析与来源判定 ----------

// certPEMInfo 是从一张 PEM 证书解析出的库存真值。
type certPEMInfo struct {
	CommonName  string
	DNSNames    []string
	IPAddresses []string
	Issuer      string
	NotBefore   time.Time
	NotAfter    time.Time
}

// PrimaryDomain 主域名取叶子证书 CN，缺省回落首个 SAN。
func (info *certPEMInfo) PrimaryDomain() string {
	if cn := strings.TrimSpace(info.CommonName); cn != "" {
		return cn
	}
	if len(info.DNSNames) > 0 {
		return info.DNSNames[0]
	}
	if len(info.IPAddresses) > 0 {
		return info.IPAddresses[0]
	}
	return ""
}

// AllDomains 全部覆盖域名（SAN 优先，CN 兜底），均小写。
func (info *certPEMInfo) AllDomains() []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(info.DNSNames)+1)
	for _, name := range info.DNSNames {
		key := caddymodel.NormalizeCertDomain(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	if cn := caddymodel.NormalizeCertDomain(info.CommonName); cn != "" {
		if _, ok := seen[cn]; !ok {
			result = append(result, cn)
		}
	}
	return result
}

// parseCertificatePEM 解析 PEM 字节中的叶子证书，取有效期/签发者/SAN。
func parseCertificatePEM(data []byte) (*certPEMInfo, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("未找到 CERTIFICATE 块")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	ips := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ips = append(ips, ip.String())
	}

	info := &certPEMInfo{
		CommonName:  cert.Subject.CommonName,
		DNSNames:    cert.DNSNames,
		IPAddresses: ips,
		Issuer:      certIssuerName(cert),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
	}
	if info.PrimaryDomain() == "" {
		return nil, fmt.Errorf("证书缺少可识别的域名")
	}
	return info, nil
}

func certIssuerName(cert *x509.Certificate) string {
	if cn := strings.TrimSpace(cert.Issuer.CommonName); cn != "" {
		return cn
	}
	if len(cert.Issuer.Organization) > 0 {
		return strings.Join(cert.Issuer.Organization, " ")
	}
	if len(cert.Issuer.OrganizationalUnit) > 0 {
		return strings.Join(cert.Issuer.OrganizationalUnit, " ")
	}
	return "未知签发者"
}

// classifyScannedCertSource 卷内证书的来源判定：
// 存储目录名带 internal/local 视为内部 CA；SAN 含泛域名只可能来自 DNS-01；
// 其余按 ACME HTTP/TLS-ALPN 处理。
func classifyScannedCertSource(issuerDir string, info *certPEMInfo) string {
	dir := strings.ToLower(issuerDir)
	if strings.Contains(dir, "internal") || strings.Contains(dir, "local") {
		return caddymodel.CertSourceInternal
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(info.CommonName)), "*.") {
		return caddymodel.CertSourceAcmeDNS
	}
	for _, name := range info.DNSNames {
		if strings.HasPrefix(strings.ToLower(name), "*.") {
			return caddymodel.CertSourceAcmeDNS
		}
	}
	return caddymodel.CertSourceAcmeHTTP
}

// ---------- 纯函数：Caddyfile 声明提取 ----------

// declaredCertInfo 是 Caddyfile 推导出的"应有证书"条目。
type declaredCertInfo struct {
	PrimaryDomain string
	Domains       []string // 站点块声明的全部域名，用于与卷内证书 SAN 求交
	Source        string
}

// extractDeclaredCerts 从 Caddyfile 提取需要证书的站点清单：
// 跳过 tls off、http:// 前缀、纯端口与 localhost 类地址；
// tls 指令形态映射：internal → internal、双路径 → manual_declared、
// tls 块内含 dns → acme_dns、缺省/其余 → acme_http（Caddy 自动 HTTPS）。
func extractDeclaredCerts(configText string) ([]declaredCertInfo, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, nil
	}
	blocks, lines, err := parseTopLevelCaddyBlocks(configText)
	if err != nil {
		return nil, err
	}

	result := make([]declaredCertInfo, 0)
	seen := make(map[string]struct{})
	for _, block := range blocks {
		if block.Kind != "site" {
			continue
		}
		domains, tlsSource, ok := siteBlockCertExpectation(lines, block)
		if !ok || len(domains) == 0 {
			continue
		}
		primary := caddymodel.NormalizeCertDomain(domains[0])
		if _, dup := seen[primary]; dup {
			continue
		}
		seen[primary] = struct{}{}
		normalized := make([]string, 0, len(domains))
		for _, domain := range domains {
			normalized = append(normalized, caddymodel.NormalizeCertDomain(domain))
		}
		result = append(result, declaredCertInfo{
			PrimaryDomain: primary,
			Domains:       normalized,
			Source:        tlsSource,
		})
	}
	return result, nil
}

// siteBlockCertExpectation 判断一个站点块是否期待证书，返回声明的域名与签发意图。
// ok 为 false 表示该站点不需要证书（tls off / 纯 http:// 地址 / 纯端口 / localhost）。
func siteBlockCertExpectation(lines []string, block caddyTopLevelBlock) (domains []string, source string, ok bool) {
	for _, token := range splitSiteAddressTokens(block.Address) {
		lower := strings.ToLower(token)
		switch {
		case strings.HasPrefix(lower, "http://"):
			// 显式 http:// 的地址不期待证书；混合地址中其余域名照常保留
			continue
		case strings.HasPrefix(lower, "https://"):
			token = token[len("https://"):]
		case strings.HasPrefix(token, ":"):
			// 纯端口地址没有可跟踪的域名
			continue
		}
		host := hostWithoutPort(token)
		if host == "" || isLocalOnlyAddress(host) || !isValidCaddySiteAddress(host) {
			continue
		}
		domains = append(domains, host)
	}

	if len(domains) == 0 {
		return nil, "", false
	}

	tlsLine, tlsBlockContent := findSiteTLSDirective(lines, block)
	if tlsLine != nil {
		if tokens := strings.Fields(*tlsLine); len(tokens) >= 2 {
			switch {
			case tokens[1] == "off":
				return nil, "", false
			case tokens[1] == "internal":
				return domains, caddymodel.CertSourceInternal, true
			case tokens[1] == "{" || strings.HasPrefix(tokens[1], "{"):
				if containsDNSChallenge(tlsBlockContent) {
					return domains, caddymodel.CertSourceAcmeDNS, true
				}
				return domains, caddymodel.CertSourceAcmeHTTP, true
			default:
				if len(tokens) >= 3 {
					return domains, caddymodel.CertSourceDeclared, true
				}
				return domains, caddymodel.CertSourceAcmeHTTP, true
			}
		}
	}
	// 无 tls 指令的域名站点走 Caddy 自动 HTTPS
	return domains, caddymodel.CertSourceAcmeHTTP, true
}

// splitSiteAddressTokens 拆站点块头地址（空格/逗号分隔的多个地址）。
func splitSiteAddressTokens(address string) []string {
	tokens := strings.FieldsFunc(address, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ','
	})
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}

func hostWithoutPort(token string) string {
	if idx := strings.LastIndex(token, ":"); idx >= 0 {
		if _, err := strconv.Atoi(token[idx+1:]); err == nil {
			return strings.TrimSpace(token[:idx])
		}
	}
	return token
}

// isLocalOnlyAddress localhost 类地址走内部解析，不纳入证书库存。
func isLocalOnlyAddress(host string) bool {
	lower := strings.ToLower(strings.TrimSpace(host))
	switch lower {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	}
	return strings.HasSuffix(lower, ".localhost") || strings.HasPrefix(lower, "127.")
}

// findSiteTLSDirective 在站点块内找 tls 指令；块形态时顺带返回块内容。
func findSiteTLSDirective(lines []string, block caddyTopLevelBlock) (tlsLine *string, blockContent string) {
	if block.StartLine < 0 || block.EndLine >= len(lines) {
		return nil, ""
	}
	for idx := block.StartLine + 1; idx < block.EndLine; idx++ {
		trimmed := strings.TrimSpace(strings.TrimRight(lines[idx], "\r\n"))
		if trimmed == "" {
			continue
		}
		tokens := strings.Fields(trimmed)
		if len(tokens) == 0 || tokens[0] != "tls" {
			continue
		}
		lineCopy := trimmed
		if len(tokens) >= 2 && tokens[1] == "{" {
			return &lineCopy, joinTLSSubBlock(lines, idx, block.EndLine)
		}
		return &lineCopy, ""
	}
	return nil, ""
}

// joinTLSSubBlock 收集 tls { 开行到配对闭括号之间的内容，用于识别 dns challenge。
func joinTLSSubBlock(lines []string, startIdx, blockEnd int) string {
	depth := 0
	var builder strings.Builder
	for idx := startIdx; idx <= blockEnd && idx < len(lines); idx++ {
		line := strings.TrimSpace(strings.TrimRight(lines[idx], "\r\n"))
		for _, ch := range line {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
			}
		}
		builder.WriteString(line)
		builder.WriteString("\n")
		if depth <= 0 {
			break
		}
	}
	return builder.String()
}

// containsDNSChallenge tls 块内出现 dns 指令即视为 DNS-01 签发意图；
// 兼容多行块与单行写法（tls { dns cloudflare }），按任意位置出现 dns 记号判断。
func containsDNSChallenge(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		for _, token := range strings.Fields(line) {
			if token == "dns" {
				return true
			}
		}
	}
	return false
}

// ---------- 纯函数：状态计算与双源合并 ----------

// certStatusAndDaysLeft 依据有效期计算状态与剩余整天数；
// 无真值（零值时间）按 pending 处理，已过期剩余天数按 0。
func certStatusAndDaysLeft(now, notAfter time.Time) (string, int) {
	if notAfter.IsZero() {
		return caddymodel.CertStatusPending, 0
	}
	if !now.Before(notAfter) {
		return caddymodel.CertStatusExpired, 0
	}
	daysLeft := int(notAfter.Sub(now).Hours() / 24)
	if daysLeft <= caddymodel.CertExpiringThresholdDays {
		return caddymodel.CertStatusExpiring, daysLeft
	}
	return caddymodel.CertStatusValid, daysLeft
}

// scannedCertInfo 是卷扫描得到的证书真值。
type scannedCertInfo struct {
	PrimaryDomain string
	Domains       []string
	Issuer        string
	Source        string
	NotBefore     time.Time
	NotAfter      time.Time
	StoragePath   string
}

// certMergeInput 双源合并的输入；Uploads 只含文件仍在盘上的上传行。
type certMergeInput struct {
	ServerID uint
	Declared []declaredCertInfo
	Scanned  []scannedCertInfo
	Uploads  []caddymodel.Certificate
	Existing []caddymodel.Certificate
	Now      time.Time
}

type certMergeResult struct {
	Upserts   []*caddymodel.Certificate
	DeleteIDs []uint
}

// mergeCertInventory 按 (serverID, 主域名) 合并三路输入为库存行集合：
//
// 优先级 上传行 > 卷扫描真值 > Caddyfile 声明；
// 声明行在无扫描命中时为 missing（本机声明了 HTTPS 但卷里没有文件）；
// 扫描/上传行按实际有效期计算 status；既有非上传行若不再出现于合并结果则下架。
// 声明域名与证书 SAN 求交匹配（apex 与 www 混用也能对上），主域名以声明侧为准。
func mergeCertInventory(input certMergeInput) certMergeResult {
	now := input.Now
	rows := make(map[string]*caddymodel.Certificate)
	order := make([]string, 0)

	upsert := func(row *caddymodel.Certificate) {
		key := caddymodel.NormalizeCertDomain(row.PrimaryDomain)
		if _, exists := rows[key]; !exists {
			order = append(order, key)
		}
		row.ServerID = input.ServerID
		row.LastSyncedAt = now
		rows[key] = row
	}

	// 1. 声明打底
	scannedByDomain := indexScannedByDomain(input.Scanned)
	usedScanned := make(map[string]struct{})
	for i := range input.Declared {
		decl := input.Declared[i]
		row := &caddymodel.Certificate{
			PrimaryDomain: decl.PrimaryDomain,
			Domains:       strings.Join(decl.Domains, ","),
			Source:        decl.Source,
			Status:        certDeclaredStatus(),
		}
		if hit := matchScanned(scannedByDomain, decl.Domains, usedScanned); hit != nil {
			fillRowFromScanned(row, hit)
			usedScanned[hit.StoragePath] = struct{}{}
		}
		if !row.NotAfter.IsZero() {
			row.Status, _ = certStatusAndDaysLeft(now, row.NotAfter)
		}
		upsert(row)
	}

	// 2. 未被声明认领的扫描行独立入库
	for i := range input.Scanned {
		scan := input.Scanned[i]
		if _, used := usedScanned[scan.StoragePath]; used {
			continue
		}
		key := caddymodel.NormalizeCertDomain(scan.PrimaryDomain)
		if _, exists := rows[key]; exists {
			continue
		}
		row := &caddymodel.Certificate{
			PrimaryDomain: scan.PrimaryDomain,
			Domains:       strings.Join(scan.Domains, ","),
			Issuer:        scan.Issuer,
			Source:        scan.Source,
			NotBefore:     scan.NotBefore,
			NotAfter:      scan.NotAfter,
			StoragePath:   scan.StoragePath,
		}
		row.Status, _ = certStatusAndDaysLeft(now, row.NotAfter)
		upsert(row)
	}

	// 3. 上传行覆盖同域合并结果（用户显式管理的证书优先展示）
	// 重定向（主域名换成 SAN 覆盖的声明域名）时须弃用原行 ID 并把旧行加入下架：
	// 带旧 ID 换主域名 upsert 会撞主键（ON CONFLICT 只认 (server_id, primary_domain)），
	// 整批落库失败且原主域名下的旧上传行永无清理机会
	redirectDeleteIDs := make([]uint, 0)
	for i := range input.Uploads {
		upload := input.Uploads[i]
		uploadKey := caddymodel.NormalizeCertDomain(upload.PrimaryDomain)
		targetKey := uploadKey
		if _, exists := rows[uploadKey]; !exists {
			// 上传证书按 SAN 覆盖声明行
			for _, domain := range splitCertDomains(upload.Domains) {
				if _, exists := rows[domain]; exists {
					targetKey = domain
					break
				}
			}
		}
		row := &upload
		row.ServerID = input.ServerID
		row.LastSyncedAt = now
		row.Status, _ = certStatusAndDaysLeft(now, row.NotAfter)
		if targetKey != uploadKey {
			if row.ID != 0 {
				redirectDeleteIDs = append(redirectDeleteIDs, row.ID)
			}
			// 置零后由第 4 步按目标键复用既有声明行 ID（如有）
			row.ID = 0
			row.PrimaryDomain = targetKey
		} else if previous, exists := rows[targetKey]; exists && previous.ID != 0 {
			row.ID = previous.ID
		}
		// 先查成员再覆盖：rows 的每个键都已在 order 里，无需再线性扫一遍
		_, wasTracked := rows[targetKey]
		rows[targetKey] = row
		if !wasTracked {
			order = append(order, targetKey)
		}
	}

	// 4. 既有行不在合并结果里的下架（上传行已在上游按文件存活过滤）；
	//    命中同域既有行时沿用其 ID，避免每次同步都删了重插
	result := certMergeResult{Upserts: make([]*caddymodel.Certificate, 0, len(rows))}
	existingByKey := make(map[string]caddymodel.Certificate)
	for i := range input.Existing {
		existing := input.Existing[i]
		if existing.Source == caddymodel.CertSourceUpload {
			continue
		}
		existingByKey[caddymodel.NormalizeCertDomain(existing.PrimaryDomain)] = existing
	}
	mergedExistingIDs := make(map[uint]struct{})
	for _, key := range order {
		row := rows[key]
		if row.ID == 0 {
			if existing, ok := existingByKey[key]; ok {
				row.ID = existing.ID
				// 去重键跟随既有行带过去：合并产物是新建对象，不带的话通知去重永远失效
				row.LastNotifiedKey = existing.LastNotifiedKey
			}
		}
		if row.ID != 0 {
			mergedExistingIDs[row.ID] = struct{}{}
		}
		result.Upserts = append(result.Upserts, row)
	}
	for i := range input.Existing {
		existing := input.Existing[i]
		if existing.Source == caddymodel.CertSourceUpload {
			continue
		}
		if _, kept := mergedExistingIDs[existing.ID]; !kept {
			result.DeleteIDs = append(result.DeleteIDs, existing.ID)
		}
	}
	// 重定向上传行的旧行（原主域名下）随本次合并一并下架
	result.DeleteIDs = append(result.DeleteIDs, redirectDeleteIDs...)
	return result
}

// certDeclaredStatus 只有声明没有真值时的状态：单实例下证书卷就在本机，
// 探不到文件即为 missing（而不是待确认）。
func certDeclaredStatus() string {
	return caddymodel.CertStatusMissing
}

func indexScannedByDomain(scanned []scannedCertInfo) map[string]*scannedCertInfo {
	index := make(map[string]*scannedCertInfo, len(scanned))
	for i := range scanned {
		scan := scanned[i]
		for _, domain := range scan.Domains {
			if _, exists := index[domain]; !exists {
				index[domain] = &scan
			}
		}
	}
	return index
}

// matchScanned 用声明域名集合与扫描索引求交，命中即认领该扫描证书。
func matchScanned(index map[string]*scannedCertInfo, domains []string, used map[string]struct{}) *scannedCertInfo {
	for _, domain := range domains {
		if hit, exists := index[domain]; exists {
			if _, isUsed := used[hit.StoragePath]; !isUsed {
				return hit
			}
		}
	}
	return nil
}

func fillRowFromScanned(row *caddymodel.Certificate, scan *scannedCertInfo) {
	row.Issuer = scan.Issuer
	row.NotBefore = scan.NotBefore
	row.NotAfter = scan.NotAfter
	row.StoragePath = scan.StoragePath
	// 扫描真值可判别 internal/dns，声明意图仅作兜底
	if scan.Source != "" {
		row.Source = scan.Source
	}
	if len(scan.Domains) > 0 {
		row.Domains = strings.Join(scan.Domains, ",")
	}
}

func splitCertDomains(joined string) []string {
	parts := strings.Split(joined, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = caddymodel.NormalizeCertDomain(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ---------- I/O：卷扫描 ----------

// scanLocalCertificates 扫描 Caddy 证书存储目录下的 PEM 证书；
// 目录不存在视为空库存（本机开发环境没有 /data/caddy），
// 单文件解析失败跳过留待下次，不中断整体扫描。
func scanLocalCertificates(root string) ([]scannedCertInfo, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("证书目录未配置")
	}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("证书路径不是目录: %s", root)
	}

	results := make([]scannedCertInfo, 0)
	seenPath := make(map[string]struct{})
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".pem") && !strings.HasSuffix(name, ".crt") {
			return nil
		}
		if name == "key.pem" || strings.HasSuffix(name, ".key") {
			return nil
		}
		fileInfo, statErr := entry.Info()
		if statErr != nil || fileInfo.Size() <= 0 || fileInfo.Size() > certScanMaxFileBytes {
			return nil
		}
		scan, scanErr := scanCertFile(root, path)
		if scanErr != nil {
			return nil
		}
		if _, dup := seenPath[scan.StoragePath]; dup {
			return nil
		}
		seenPath[scan.StoragePath] = struct{}{}
		results = append(results, *scan)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// scanCertFile 解析单个证书文件并按相对路径判定来源：
// 布局为 <root>/<签发者目录>/<域名目录>/cert.pem，签发者目录取首段。
func scanCertFile(root, path string) (*scannedCertInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	info, err := parseCertificatePEM(data)
	if err != nil {
		return nil, err
	}

	issuerDir := ""
	if rel, relErr := filepath.Rel(root, path); relErr == nil {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) > 1 {
			issuerDir = parts[0]
		}
	}

	return &scannedCertInfo{
		PrimaryDomain: info.PrimaryDomain(),
		Domains:       info.AllDomains(),
		Issuer:        info.Issuer,
		Source:        classifyScannedCertSource(issuerDir, info),
		NotBefore:     info.NotBefore,
		NotAfter:      info.NotAfter,
		StoragePath:   filepath.Clean(path),
	}, nil
}

// certUploadDir 上传证书的落盘目录：/config/certs/<主域名>。
//
// 单实例不再按节点分目录，存量 /config/certs/<serverID>/<域名>/ 里的文件
// 由证书库存的 StoragePath 定位，仍可正常删除。
func certUploadDir(primaryDomain string) string {
	return filepath.Join(certUploadBaseDir, certStorageDirName(primaryDomain))
}

// certStorageDirName 域名转目录名，剔除路径分隔符与通配符等危险字符。
func certStorageDirName(domain string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", " ", "_", "..", "_", "*", "_",
	)
	cleaned := replacer.Replace(strings.TrimSpace(domain))
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}

// resolveLocalCertServer 返回上传证书的落点（唯一实例，恒为本机）。
func resolveLocalCertServer(db *gorm.DB) (*caddymodel.CaddyServer, error) {
	server, err := findPreferredCaddyServer(db)
	if err != nil {
		return nil, xerr.NewBusinessErrorWith(err.Error())
	}
	return server, nil
}
