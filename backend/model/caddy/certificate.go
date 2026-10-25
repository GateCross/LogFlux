package caddy

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 证书来源取值，长度上限 32，新增取值不得超出。
const (
	CertSourceAcmeHTTP = "acme_http"       // Caddy 自动签发（HTTP-01/TLS-ALPN）
	CertSourceAcmeDNS  = "acme_dns"        // Caddy 自动签发（DNS-01，如 Cloudflare）
	CertSourceInternal = "internal"        // Caddy 内部 CA
	CertSourceUpload   = "upload"          // 用户经证书中心上传
	CertSourceDeclared = "manual_declared" // Caddyfile 声明的手动证书路径，仅有声明无真值
)

// 证书状态取值，长度上限 16，新增取值不得超出。
const (
	CertStatusValid    = "valid"    // 在有效期且距到期超过阈值
	CertStatusExpiring = "expiring" // 距到期不超过 14 天
	CertStatusExpired  = "expired"  // 已过期
	CertStatusMissing  = "missing"  // 本机声明了 HTTPS 但卷里没有对应证书文件
	CertStatusPending  = "pending"  // 远端服务器仅有 Caddyfile 声明，无真值
)

// 到期告警阈值：距到期不超过该天数即视为 expiring。
const CertExpiringThresholdDays = 14

// Certificate 是证书中心的库存行，由卷扫描与 Caddyfile 声明两源合并生成；
// (server_id, primary_domain) 是合并键，source=upload 的行由用户上传维护。
type Certificate struct {
	ID            uint `gorm:"primarykey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ServerID      uint      `gorm:"not null;uniqueIndex:uk_caddy_cert_server_domain,priority:1"`
	PrimaryDomain string    `gorm:"size:255;not null;uniqueIndex:uk_caddy_cert_server_domain,priority:2"`
	Domains       string    `gorm:"type:text"` // SAN 列表，逗号分隔
	Issuer        string    `gorm:"size:255"`  // 签发者展示名
	Source        string    `gorm:"size:32;not null;default:'manual_declared'"`
	NotBefore     time.Time `gorm:"default:NULL"` // 零值表示无真值（声明行）
	NotAfter      time.Time `gorm:"default:NULL"` // 零值表示无真值（声明行）
	StoragePath   string    `gorm:"type:text"`    // 卷内路径或 /config/certs 上传路径
	Status        string    `gorm:"size:16;not null;default:'pending'"`
	LastSyncedAt  time.Time `gorm:"default:NULL"`
	// LastNotifiedKey 最近一次到期通知的去重键（状态:级别:到期时间戳）。
	// 同键不重发：每日同步与手动刷新重复触发时静默，状态跃迁或证书续期换键后恢复提醒。
	LastNotifiedKey string `gorm:"size:64;not null;default:''"`
}

func (Certificate) TableName() string {
	return "caddy_certificates"
}

// CertificateQuery 库存查询条件，ServerID 为 0 表示全部服务器。
type CertificateQuery struct {
	ServerID uint
}

type CertificateModel interface {
	List(ctx context.Context, query CertificateQuery) ([]Certificate, error)
	FindByID(ctx context.Context, id uint) (*Certificate, error)
	UpsertBatch(ctx context.Context, rows []*Certificate) error
	DeleteByIDs(ctx context.Context, ids []uint) error
	// UpdateLastNotifiedKey 记录单张证书最近一次到期通知的去重键。
	UpdateLastNotifiedKey(ctx context.Context, id uint, key string) error
}

type defaultCertificateModel struct {
	db *gorm.DB
}

func NewCertificateModel(db *gorm.DB) CertificateModel {
	return &defaultCertificateModel{db: db}
}

func (m *defaultCertificateModel) conn(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return m.db.WithContext(ctx)
}

// List 按服务器过滤库存，按服务器与主域名排序保证输出稳定。
func (m *defaultCertificateModel) List(ctx context.Context, query CertificateQuery) ([]Certificate, error) {
	db := m.conn(ctx).Order("server_id asc, primary_domain asc")
	if query.ServerID > 0 {
		db = db.Where("server_id = ?", query.ServerID)
	}
	var rows []Certificate
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultCertificateModel) FindByID(ctx context.Context, id uint) (*Certificate, error) {
	var row Certificate
	if err := m.conn(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UpsertBatch 按 (server_id, primary_domain) 冲突时更新合并结果列，
// 供双源同步与上传共用。
func (m *defaultCertificateModel) UpsertBatch(ctx context.Context, rows []*Certificate) error {
	if len(rows) == 0 {
		return nil
	}
	return m.conn(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "server_id"}, {Name: "primary_domain"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"domains", "issuer", "source", "not_before", "not_after",
			"storage_path", "status", "last_synced_at", "updated_at",
		}),
	}).Create(rows).Error
}

func (m *defaultCertificateModel) DeleteByIDs(ctx context.Context, ids []uint) error {
	ids = compactCertIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	return m.conn(ctx).Where("id IN ?", ids).Delete(&Certificate{}).Error
}

// UpdateLastNotifiedKey 只更新去重键列，不触碰库存同步列。
func (m *defaultCertificateModel) UpdateLastNotifiedKey(ctx context.Context, id uint, key string) error {
	if id == 0 {
		return nil
	}
	return m.conn(ctx).Model(&Certificate{}).Where("id = ?", id).
		Update("last_notified_key", key).Error
}

func compactCertIDs(ids []uint) []uint {
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			result = append(result, id)
		}
	}
	return result
}

// NormalizeCertDomain 域名统一小写并去空白，作为合并键。
func NormalizeCertDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}
