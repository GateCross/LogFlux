package caddy

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"

	ingestmodel "logflux/model/ingest"
	systemmodel "logflux/model/system"

	"github.com/zeromicro/go-zero/core/logx"
)

// 单实例改造后当前生效配置的存储位置。
//
// 配置只有一份，与入口块、日志 snippet 等单例配置同类，因此存 system_configs；
// caddy_servers 表只作为迁移来源保留，不再读写。
const (
	// ConfigKey 当前生效 Caddy 配置在 system_configs 里的键。
	ConfigKey = "caddy_config"
	// ConfigMigratedKey 迁移完成标记，便于排查与幂等判断。
	ConfigMigratedKey = "caddy_config_migrated"
)

// EmptyModulesJSON 是结构化配置为空时的占位值（jsonb 列不接受空串）。
const EmptyModulesJSON = "{}"

// ConfigValue 当前生效配置的存储结构。
type ConfigValue struct {
	Config  string `json:"config"`
	Modules string `json:"modules"`
}

// LoadConfig 读取当前生效配置；键不存在或内容非法时返回空配置（首次部署）。
// 返回的 modules 可能是空串，调用方按需归一化。
func LoadConfig(db *gorm.DB) (string, string) {
	if db == nil {
		return "", ""
	}
	cfg, err := systemmodel.NewSystemConfigModel(db).GetByKey(ConfigKey)
	if err != nil {
		return "", ""
	}
	var value ConfigValue
	if err := json.Unmarshal([]byte(cfg.Value), &value); err != nil {
		return "", ""
	}
	if strings.TrimSpace(value.Config) == "" {
		return "", ""
	}
	return value.Config, value.Modules
}

// SaveConfig 写入当前生效配置。
func SaveConfig(db *gorm.DB, config, modules string) error {
	if strings.TrimSpace(modules) == "" {
		modules = EmptyModulesJSON
	}
	encoded, err := json.Marshal(ConfigValue{Config: config, Modules: modules})
	if err != nil {
		return err
	}
	return systemmodel.NewSystemConfigModel(db).SetByKey(ConfigKey, string(encoded))
}

// MigrateLegacyConfig 把 caddy_servers 上的配置搬迁到 system_configs，
// 并把仍带 server_id 的表统一归零，供单实例改造升级使用。
//
// 幂等：已有 caddy_config 键即跳过搬迁；server_id 归零重复执行结果相同。
// caddy_servers 表不删，迁移失败仍可回滚到旧版本。
func MigrateLegacyConfig(db *gorm.DB) {
	if db == nil {
		return
	}
	migrateConfigValue(db)
	zeroLegacyServerIDs(db)
	if err := systemmodel.NewSystemConfigModel(db).SetByKey(ConfigMigratedKey, "true"); err != nil {
		logx.Errorf("写入 Caddy 配置迁移标记失败: %v", err)
	}
}

// migrateConfigValue 取 caddy_servers 里 id 最小一行的配置写入 system_configs。
func migrateConfigValue(db *gorm.DB) {
	if _, err := systemmodel.NewSystemConfigModel(db).GetByKey(ConfigKey); err == nil {
		return
	}
	if !db.Migrator().HasTable(&CaddyServer{}) {
		return
	}

	var servers []CaddyServer
	if err := db.Order("id asc").Find(&servers).Error; err != nil {
		logx.Errorf("迁移 Caddy 配置失败，读取节点列表出错: %v", err)
		return
	}
	if len(servers) == 0 {
		return
	}
	// 老部署理论上只有一个节点；多于一个时只保留 id 最小的配置，其余告警但不删数据。
	if len(servers) > 1 {
		logx.Errorf(
			"检测到 %d 个 Caddy 节点，单实例模式只保留 id=%d 的配置，其余节点的配置历史与证书库存将失去关联，请手动备份",
			len(servers), servers[0].ID,
		)
	}
	if strings.TrimSpace(servers[0].Config) == "" {
		return
	}
	if err := SaveConfig(db, servers[0].Config, servers[0].Modules); err != nil {
		logx.Errorf("迁移 Caddy 配置到 system_configs 失败: %v", err)
		return
	}
	logx.Infof("已迁移 Caddy 配置到 system_configs: 来源节点 id=%d", servers[0].ID)
}

// zeroLegacyServerIDs 把历史数据里的 server_id 统一归零，避免迁移后历史断代：
// 新代码按 server_id = 0 读写，旧数据是 1，不归零就查不到旧记录。
func zeroLegacyServerIDs(db *gorm.DB) {
	targets := []struct {
		model any
		label string
	}{
		{&CaddyConfigHistory{}, "配置历史"},
		{&Certificate{}, "证书库存"},
		{&ingestmodel.LogSource{}, "日志源"},
	}
	for _, target := range targets {
		if !db.Migrator().HasTable(target.model) {
			continue
		}
		result := db.Model(target.model).Where("server_id <> 0").Update("server_id", 0)
		if result.Error != nil {
			logx.Errorf("归零%s的 server_id 失败: %v", target.label, result.Error)
			continue
		}
		if result.RowsAffected > 0 {
			logx.Infof("已归零%s的 server_id: %d 行", target.label, result.RowsAffected)
		}
	}
}
