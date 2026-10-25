package caddy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"logflux/internal/notification"
	"logflux/internal/svc"

	caddymodel "logflux/model/caddy"
)

// generateTestCertPair 现场生成自签证书与私钥 PEM，供纯函数测试使用。
func generateTestCertPair(t *testing.T, commonName string, dnsNames []string, notBefore, notAfter time.Time) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"LogFlux 测试 CA"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		DNSNames:              dnsNames,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("签发测试证书失败: %v", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("序列化测试私钥失败: %v", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestParseCertificatePEM(t *testing.T) {
	now := time.Now()
	certPEM, keyPEM := generateTestCertPair(t, "a.example.com", []string{"a.example.com", "b.example.com"}, now.Add(-time.Hour), now.Add(90*24*time.Hour))

	info, err := parseCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("解析证书失败: %v", err)
	}
	if info.CommonName != "a.example.com" {
		t.Errorf("CN 期望 a.example.com, 实际 %s", info.CommonName)
	}
	if info.PrimaryDomain() != "a.example.com" {
		t.Errorf("主域名期望 a.example.com, 实际 %s", info.PrimaryDomain())
	}
	if len(info.AllDomains()) != 2 {
		t.Errorf("SAN 域名数期望 2, 实际 %d (%v)", len(info.AllDomains()), info.AllDomains())
	}
	if info.Issuer == "" || info.Issuer == "未知签发者" {
		t.Errorf("签发者不应为空: %q", info.Issuer)
	}
	if !info.NotAfter.After(info.NotBefore) {
		t.Errorf("有效期区间异常: %v ~ %v", info.NotBefore, info.NotAfter)
	}

	// 私钥 PEM 不是证书，必须报错
	if _, err := parseCertificatePEM(keyPEM); err == nil {
		t.Error("私钥 PEM 不应被解析为证书")
	}
	// 垃圾输入报错
	if _, err := parseCertificatePEM([]byte("not a pem")); err == nil {
		t.Error("垃圾输入应报错")
	}
}

func TestParseCertificatePEMNoDomain(t *testing.T) {
	now := time.Now()
	certPEM, _ := generateTestCertPair(t, "", nil, now, now.Add(time.Hour))
	if _, err := parseCertificatePEM(certPEM); err == nil {
		t.Error("无 CN 且无 SAN 的证书应报错")
	}
}

func TestCertStatusAndDaysLeft(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		notAfter time.Time
		status   string
		daysLeft int
	}{
		{"零值时间按 pending", time.Time{}, caddymodel.CertStatusPending, 0},
		{"已过期", now.Add(-time.Hour), caddymodel.CertStatusExpired, 0},
		{"剩 1 天为 expiring", now.Add(24 * time.Hour), caddymodel.CertStatusExpiring, 1},
		{"剩 7 天为 expiring", now.Add(7 * 24 * time.Hour), caddymodel.CertStatusExpiring, 7},
		{"剩 14 天为 expiring", now.Add(14 * 24 * time.Hour), caddymodel.CertStatusExpiring, 14},
		{"剩 15 天为 valid", now.Add(15 * 24 * time.Hour), caddymodel.CertStatusValid, 15},
		{"剩 90 天为 valid", now.Add(90 * 24 * time.Hour), caddymodel.CertStatusValid, 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, daysLeft := certStatusAndDaysLeft(now, tc.notAfter)
			if status != tc.status {
				t.Errorf("状态期望 %s, 实际 %s", tc.status, status)
			}
			if daysLeft != tc.daysLeft {
				t.Errorf("剩余天数期望 %d, 实际 %d", tc.daysLeft, daysLeft)
			}
		})
	}
}

func TestExtractDeclaredCerts(t *testing.T) {
	configText := `{
  order coraza_waf first
}

(waf_protect) {
  coraza_waf {
    directives ` + "`SecRuleEngine On`" + `
  }
}

example.com {
  tls internal
  reverse_proxy localhost:8080
}

auto.example.com {
  reverse_proxy localhost:8081
}

manual.example.com {
  tls /config/certs/1/manual.example.com/fullchain.pem /config/certs/1/manual.example.com/privkey.pem
}

dns.example.com {
  tls {
    dns cloudflare
  }
  reverse_proxy localhost:8082
}

dns-inline.example.com {
  tls { dns cloudflare }
}

off.example.com {
  tls off
}

http://plain.example.com {
  respond 200
}

http-mixed.example.com http://skip.example.com {
  respond 200
}

:80 {
  respond 200
}

localhost {
  respond 200
}

multi.example.com, alt.example.com {
  reverse_proxy localhost:8083
}

withport.example.com:8443 {
  reverse_proxy localhost:8084
}
`
	declared, err := extractDeclaredCerts(configText)
	if err != nil {
		t.Fatalf("提取声明失败: %v", err)
	}

	got := make(map[string]string, len(declared))
	for _, item := range declared {
		got[item.PrimaryDomain] = item.Source
	}
	expect := map[string]string{
		"example.com":            caddymodel.CertSourceInternal,
		"auto.example.com":       caddymodel.CertSourceAcmeHTTP,
		"manual.example.com":     caddymodel.CertSourceDeclared,
		"dns.example.com":        caddymodel.CertSourceAcmeDNS,
		"dns-inline.example.com": caddymodel.CertSourceAcmeDNS,
		"multi.example.com":      caddymodel.CertSourceAcmeHTTP,
		"withport.example.com":   caddymodel.CertSourceAcmeHTTP,
		"http-mixed.example.com": caddymodel.CertSourceAcmeHTTP,
	}
	for domain, source := range expect {
		if got[domain] != source {
			t.Errorf("域名 %s 来源期望 %s, 实际 %s", domain, source, got[domain])
		}
	}
	// 不需要证书的站点不应出现
	for _, domain := range []string{"off.example.com", "plain.example.com", "skip.example.com", "localhost"} {
		if _, exists := got[domain]; exists {
			t.Errorf("域名 %s 不应出现在声明清单", domain)
		}
	}
	// alt 域名应挂在 multi 主域名条目下
	for _, item := range declared {
		if item.PrimaryDomain == "multi.example.com" {
			if len(item.Domains) != 2 || item.Domains[1] != "alt.example.com" {
				t.Errorf("multi 站点应携带 alt 域名, 实际 %v", item.Domains)
			}
		}
	}
}

func TestClassifyScannedCertSource(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		issuerDir string
		cn        string
		dnsNames  []string
		source    string
	}{
		{"内部 CA 目录", "internal", "a.com", []string{"a.com"}, caddymodel.CertSourceInternal},
		{"local 目录", "local", "a.com", []string{"a.com"}, caddymodel.CertSourceInternal},
		{"ACME 目录普通域名", "acme-v02.api.letsencrypt.org-directory", "a.com", []string{"a.com"}, caddymodel.CertSourceAcmeHTTP},
		{"ACME 目录泛域名", "acme-v02.api.letsencrypt.org-directory", "*.a.com", []string{"*.a.com"}, caddymodel.CertSourceAcmeDNS},
		{"SAN 泛域名", "acme.zerossl.com-v2-dv90", "a.com", []string{"a.com", "*.a.com"}, caddymodel.CertSourceAcmeDNS},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			certPEM, _ := generateTestCertPair(t, tc.cn, tc.dnsNames, now, now.Add(time.Hour))
			info, err := parseCertificatePEM(certPEM)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if source := classifyScannedCertSource(tc.issuerDir, info); source != tc.source {
				t.Errorf("来源期望 %s, 实际 %s", tc.source, source)
			}
		})
	}
}

func TestMergeCertInventory(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	healthyCert, _ := generateTestCertPair(t, "ok.example.com", []string{"ok.example.com"}, now.Add(-24*time.Hour), now.Add(90*24*time.Hour))
	expiringCert, _ := generateTestCertPair(t, "soon.example.com", []string{"soon.example.com"}, now.Add(-24*time.Hour), now.Add(5*24*time.Hour))
	healthyInfo, err := parseCertificatePEM(healthyCert)
	if err != nil {
		t.Fatalf("解析健康证书失败: %v", err)
	}
	expiringInfo, err := parseCertificatePEM(expiringCert)
	if err != nil {
		t.Fatalf("解析临期证书失败: %v", err)
	}

	t.Run("本机声明+扫描命中合并为真值", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "ok.example.com", Domains: []string{"ok.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Scanned: []scannedCertInfo{{
				PrimaryDomain: "ok.example.com",
				Domains:       []string{"ok.example.com"},
				Issuer:        healthyInfo.Issuer,
				Source:        caddymodel.CertSourceAcmeHTTP,
				NotBefore:     healthyInfo.NotBefore,
				NotAfter:      healthyInfo.NotAfter,
				StoragePath:   "/data/caddy/certificates/acme/ok.example.com/cert.pem",
			}},
			Now: now,
		})
		if len(result.Upserts) != 1 || len(result.DeleteIDs) != 0 {
			t.Fatalf("合并行数期望 1/0, 实际 %d/%d", len(result.Upserts), len(result.DeleteIDs))
		}
		row := result.Upserts[0]
		if row.Status != caddymodel.CertStatusValid || row.NotAfter.IsZero() || row.StoragePath == "" {
			t.Errorf("命中扫描后应带真值: %+v", row)
		}
		if row.Source != caddymodel.CertSourceAcmeHTTP {
			t.Errorf("来源期望 acme_http, 实际 %s", row.Source)
		}
	})

	t.Run("本机声明无扫描命中为 missing", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "gap.example.com", Domains: []string{"gap.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Now:        now,
		})
		if len(result.Upserts) != 1 {
			t.Fatalf("合并行数期望 1, 实际 %d", len(result.Upserts))
		}
		row := result.Upserts[0]
		if row.Status != caddymodel.CertStatusMissing || !row.NotAfter.IsZero() {
			t.Errorf("本机缺口行应为 missing 且有效期零值: %+v", row)
		}
	})

	t.Run("远端声明恒为 pending", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   2,
			ServerType: "remote",
			Declared:   []declaredCertInfo{{PrimaryDomain: "far.example.com", Domains: []string{"far.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Now:        now,
		})
		row := result.Upserts[0]
		if row.Status != caddymodel.CertStatusPending || !row.NotAfter.IsZero() {
			t.Errorf("远端声明行应为 pending 且有效期零值: %+v", row)
		}
	})

	t.Run("上传行覆盖同域合并结果并沿用 ID", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "up.example.com", Domains: []string{"up.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Uploads: []caddymodel.Certificate{{
				ID:            77,
				ServerID:      1,
				PrimaryDomain: "up.example.com",
				Domains:       "up.example.com",
				Source:        caddymodel.CertSourceUpload,
				NotBefore:     expiringInfo.NotBefore,
				NotAfter:      expiringInfo.NotAfter,
				StoragePath:   "/config/certs/1/up.example.com/fullchain.pem",
			}},
			Now: now,
		})
		if len(result.Upserts) != 1 {
			t.Fatalf("合并行数期望 1, 实际 %d", len(result.Upserts))
		}
		row := result.Upserts[0]
		if row.Source != caddymodel.CertSourceUpload || row.ID != 77 {
			t.Errorf("上传行应覆盖并沿用 ID: %+v", row)
		}
		if row.Status != caddymodel.CertStatusExpiring {
			t.Errorf("临期上传证书状态期望 expiring, 实际 %s", row.Status)
		}
		if len(result.DeleteIDs) != 0 {
			t.Errorf("不应产生下架: %v", result.DeleteIDs)
		}
	})

	t.Run("上传行按 SAN 重定向到声明域名：弃用旧 ID、沿用目标行 ID、下架旧行", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "site.example.com", Domains: []string{"site.example.com"}, Source: caddymodel.CertSourceDeclared}},
			Uploads: []caddymodel.Certificate{{
				ID:            88,
				ServerID:      1,
				PrimaryDomain: "certcn.example.com",
				Domains:       "certcn.example.com,site.example.com",
				Source:        caddymodel.CertSourceUpload,
				NotBefore:     expiringInfo.NotBefore,
				NotAfter:      expiringInfo.NotAfter,
				StoragePath:   "/config/certs/1/certcn.example.com/fullchain.pem",
			}},
			Existing: []caddymodel.Certificate{
				{ID: 50, ServerID: 1, PrimaryDomain: "site.example.com", Source: caddymodel.CertSourceDeclared},
				{ID: 88, ServerID: 1, PrimaryDomain: "certcn.example.com", Source: caddymodel.CertSourceUpload},
			},
			Now: now,
		})
		if len(result.Upserts) != 1 {
			t.Fatalf("合并行数期望 1, 实际 %d", len(result.Upserts))
		}
		row := result.Upserts[0]
		if row.PrimaryDomain != "site.example.com" || row.Source != caddymodel.CertSourceUpload {
			t.Errorf("重定向行应为上传来源且主域名为声明域名: %+v", row)
		}
		if row.ID == 88 {
			t.Errorf("重定向行不得携带原主域名下的旧库 ID，换键 upsert 会撞主键: %+v", row)
		}
		if row.ID != 50 {
			t.Errorf("目标键既有声明行 ID 期望被沿用: %+v", row)
		}
		if len(result.DeleteIDs) != 1 || result.DeleteIDs[0] != 88 {
			t.Fatalf("原主域名下的旧上传行期望下架 [88], 实际 %v", result.DeleteIDs)
		}
	})

	t.Run("卷里消失的本机证书下架", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "kept.example.com", Domains: []string{"kept.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Existing: []caddymodel.Certificate{
				{ID: 11, ServerID: 1, PrimaryDomain: "kept.example.com", Source: caddymodel.CertSourceAcmeHTTP},
				{ID: 12, ServerID: 1, PrimaryDomain: "gone.example.com", Source: caddymodel.CertSourceAcmeHTTP},
			},
			Now: now,
		})
		if len(result.DeleteIDs) != 1 || result.DeleteIDs[0] != 12 {
			t.Fatalf("下架期望 [12], 实际 %v", result.DeleteIDs)
		}
		// 存留行沿用既有 ID，避免每日同步删了重插
		var kept *caddymodel.Certificate
		for _, row := range result.Upserts {
			if row.PrimaryDomain == "kept.example.com" {
				kept = row
			}
		}
		if kept == nil || kept.ID != 11 {
			t.Errorf("存留行应沿用 ID 11: %+v", kept)
		}
	})

	t.Run("声明域名与证书 SAN 求交匹配", func(t *testing.T) {
		// 站点声明 www2.example.com，证书 CN/SAN 是 apex + www2
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Declared:   []declaredCertInfo{{PrimaryDomain: "www2.example.com", Domains: []string{"www2.example.com"}, Source: caddymodel.CertSourceAcmeHTTP}},
			Scanned: []scannedCertInfo{{
				PrimaryDomain: "apex.example.com",
				Domains:       []string{"apex.example.com", "www2.example.com"},
				Issuer:        healthyInfo.Issuer,
				Source:        caddymodel.CertSourceAcmeHTTP,
				NotBefore:     healthyInfo.NotBefore,
				NotAfter:      healthyInfo.NotAfter,
				StoragePath:   "/data/caddy/certificates/acme/apex.example.com/cert.pem",
			}},
			Now: now,
		})
		if len(result.Upserts) != 1 {
			t.Fatalf("声明与扫描应合并为一行, 实际 %d", len(result.Upserts))
		}
		row := result.Upserts[0]
		if row.PrimaryDomain != "www2.example.com" {
			t.Errorf("主域名应以声明侧为准, 实际 %s", row.PrimaryDomain)
		}
		if row.Status != caddymodel.CertStatusValid {
			t.Errorf("命中真值后状态期望 valid, 实际 %s", row.Status)
		}
	})

	t.Run("未被声明认领的扫描行独立入库", func(t *testing.T) {
		result := mergeCertInventory(certMergeInput{
			ServerID:   1,
			ServerType: "local",
			Scanned: []scannedCertInfo{{
				PrimaryDomain: "orphan.example.com",
				Domains:       []string{"orphan.example.com"},
				Issuer:        healthyInfo.Issuer,
				Source:        caddymodel.CertSourceAcmeHTTP,
				NotBefore:     healthyInfo.NotBefore,
				NotAfter:      healthyInfo.NotAfter,
				StoragePath:   "/data/caddy/certificates/acme/orphan.example.com/cert.pem",
			}},
			Now: now,
		})
		if len(result.Upserts) != 1 || result.Upserts[0].PrimaryDomain != "orphan.example.com" {
			t.Fatalf("扫描行应独立入库: %+v", result.Upserts)
		}
	})
}

func TestScanLocalCertificates(t *testing.T) {
	t.Run("目录不存在视为空库存", func(t *testing.T) {
		results, err := scanLocalCertificates(t.TempDir() + "/not-exists")
		if err != nil || len(results) != 0 {
			t.Fatalf("期望空库存且无错误, 实际 %v %v", results, err)
		}
	})

	t.Run("按布局扫描并跳过私钥", func(t *testing.T) {
		now := time.Now()
		root := t.TempDir()
		certPEM, keyPEM := generateTestCertPair(t, "scan.example.com", []string{"scan.example.com"}, now.Add(-time.Hour), now.Add(30*24*time.Hour))
		certDir := root + "/acme-v02.api.letsencrypt.org-directory/scan.example.com"
		if err := os.MkdirAll(certDir, 0o755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(certDir+"/cert.pem", certPEM, 0o600); err != nil {
			t.Fatalf("写入证书失败: %v", err)
		}
		if err := os.WriteFile(certDir+"/key.pem", keyPEM, 0o600); err != nil {
			t.Fatalf("写入私钥失败: %v", err)
		}

		results, err := scanLocalCertificates(root)
		if err != nil {
			t.Fatalf("扫描失败: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("扫描结果期望 1 张证书, 实际 %d", len(results))
		}
		scan := results[0]
		if scan.PrimaryDomain != "scan.example.com" {
			t.Errorf("主域名期望 scan.example.com, 实际 %s", scan.PrimaryDomain)
		}
		if scan.Source != caddymodel.CertSourceAcmeHTTP {
			t.Errorf("来源期望 acme_http, 实际 %s", scan.Source)
		}
		if scan.NotAfter.IsZero() {
			t.Error("扫描结果应带有效期真值")
		}
	})

	t.Run("解析失败的文件跳过不中断", func(t *testing.T) {
		root := t.TempDir()
		certDir := root + "/issuer/bad.example.com"
		if err := os.MkdirAll(certDir, 0o755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		if err := os.WriteFile(certDir+"/cert.pem", []byte("-----BEGIN CERTIFICATE-----\nbroken\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
			t.Fatalf("写入坏证书失败: %v", err)
		}
		results, err := scanLocalCertificates(root)
		if err != nil {
			t.Fatalf("坏文件应被跳过而不是报错: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("坏文件不应入库, 实际 %d 条", len(results))
		}
	})
}

func TestCertStorageDirName(t *testing.T) {
	cases := map[string]string{
		"example.com":     "example.com",
		"*.example.com":   "_.example.com",
		"a b.example.com": "a_b.example.com",
		"..\\evil":        "__evil",
		"":                "unknown",
	}
	for input, expect := range cases {
		if got := certStorageDirName(input); got != expect {
			t.Errorf("目录名 %q 期望 %q, 实际 %q", input, expect, got)
		}
	}
}

// fakeCertNotifyManager 只覆写 Notify 的通知管理器桩，记录收到的事件。
type fakeCertNotifyManager struct {
	notification.NotificationManager
	events []*notification.Event
}

func (m *fakeCertNotifyManager) Notify(_ context.Context, event *notification.Event) error {
	m.events = append(m.events, event)
	return nil
}

// fakeCertNotifyModel 只覆写 UpdateLastNotifiedKey 的证书模型桩。
type fakeCertNotifyModel struct {
	caddymodel.CertificateModel
	keys map[uint]string
}

func (m *fakeCertNotifyModel) UpdateLastNotifiedKey(_ context.Context, id uint, key string) error {
	m.keys[id] = key
	return nil
}

// TestNotifyCertStateDedupedByNotifiedKey 锁定到期通知去重：同一「状态:级别:到期时间」
// 组合只发一次，每日同步与手动刷新重复触发不再轰炸渠道；
// 状态跃迁（临期→过期）或续期换键后恢复提醒。
func TestNotifyCertStateDedupedByNotifiedKey(t *testing.T) {
	now := time.Now()
	mgr := &fakeCertNotifyManager{}
	model := &fakeCertNotifyModel{keys: map[uint]string{}}
	svcCtx := &svc.ServiceContext{NotificationMgr: mgr, CertificateModel: model}
	service := NewCertSyncService(context.Background(), svcCtx)
	server := &caddymodel.CaddyServer{ID: 1, Name: "local-default"}

	row := &caddymodel.Certificate{
		ID:            7,
		ServerID:      1,
		PrimaryDomain: "soon.example.com",
		NotAfter:      now.Add(5 * 24 * time.Hour),
	}

	// 第一轮同步：临期 warning 发一次并落去重键
	service.notifyCertState(server, []*caddymodel.Certificate{row})
	if len(mgr.events) != 1 {
		t.Fatalf("临期证书应发 1 次通知, 实际 %d", len(mgr.events))
	}
	key, ok := model.keys[row.ID]
	if !ok || key == "" {
		t.Fatalf("发送成功后应落去重键, 实际 %+v", model.keys)
	}

	// 第二轮同步（模拟手动刷新）：同键已标记，静默跳过
	row.LastNotifiedKey = key
	service.notifyCertState(server, []*caddymodel.Certificate{row})
	if len(mgr.events) != 1 {
		t.Fatalf("同键重复同步不应重发, 实际 %d 次", len(mgr.events))
	}

	// 状态跃迁为已过期：换键恢复提醒
	row.NotAfter = now.Add(-time.Hour)
	service.notifyCertState(server, []*caddymodel.Certificate{row})
	if len(mgr.events) != 2 {
		t.Fatalf("过期跃迁应再发 1 次, 实际 %d 次", len(mgr.events))
	}
	if mgr.events[1].Type != notification.EventCertExpired {
		t.Fatalf("第二次应为期满事件, 实际 %s", mgr.events[1].Type)
	}

	// 续期后再次临期：到期时间变化换键，恢复提醒
	row.NotAfter = now.Add(10 * 24 * time.Hour)
	row.LastNotifiedKey = model.keys[row.ID]
	service.notifyCertState(server, []*caddymodel.Certificate{row})
	if len(mgr.events) != 3 {
		t.Fatalf("续期后再次临期应恢复提醒, 实际 %d 次", len(mgr.events))
	}
}
