package caddy

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newConfigStoreMockDB 建 sqlmock 驱动的 GORM 句柄（postgres 方言）。
func newConfigStoreMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqldb}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		_ = sqldb.Close()
		t.Fatalf("gorm: %v", err)
	}
	return gdb, mock, func() { _ = sqldb.Close() }
}

func configQueryRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"})
}

func configRows(key, value string) *sqlmock.Rows {
	return configQueryRows().AddRow(uint(1), time.Now(), time.Now(), key, value)
}

// expectHasTable 期望一次 HasTable 探测；exists 决定后续是否继续操作该表。
func expectHasTable(mock sqlmock.Sqlmock, exists bool) {
	count := int64(0)
	if exists {
		count = 1
	}
	mock.ExpectQuery(`SELECT count\(\*\) FROM information_schema\.tables`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

// expectConfigWrite 期望一次 SetByKey：先查键，查不到再插入。
func expectConfigWrite(mock sqlmock.Sqlmock, key string) {
	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(configQueryRows())
	mock.ExpectQuery(`INSERT INTO "system_configs"`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(9)))
}

// 读写往返：写入后能原样读回。
func TestCaddyConfigStoreRoundTrip(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(configRows(ConfigKey, `{"config":"a","modules":"{}"}`))
	config, modules := LoadConfig(db)
	if config != "a" || modules != "{}" {
		t.Fatalf("往返读取不一致: config=%q modules=%q", config, modules)
	}

	expectConfigWrite(mock, ConfigKey)
	if err := SaveConfig(db, "b", ""); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql 期望未满足: %v", err)
	}
}

// 键不存在时返回空配置，由调用方按「首次部署」处理。
func TestLoadConfigMissingKeyReturnsEmpty(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(configQueryRows())

	config, modules := LoadConfig(db)
	if config != "" || modules != "" {
		t.Fatalf("键不存在应返回空值: config=%q modules=%q", config, modules)
	}
}

// 已有 caddy_config 键时迁移必须跳过搬迁，只做 server_id 归零与标记写入。
func TestMigrateLegacyConfigSkipsWhenConfigExists(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(configRows(ConfigKey, `{"config":"existing","modules":"{}"}`))
	// 三张表：HasTable + UPDATE
	for i := 0; i < 3; i++ {
		expectHasTable(mock, true)
		mock.ExpectExec(`UPDATE .* SET .*server_id`).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	expectConfigWrite(mock, ConfigMigratedKey)

	MigrateLegacyConfig(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("已有配置键时不应搬迁: %v", err)
	}
}

// 老库单节点：把该行配置搬进 system_configs，并把三张表的历史 server_id 归零。
func TestMigrateLegacyConfigCopiesSingleNodeConfig(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(configQueryRows())
	expectHasTable(mock, true)
	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "url", "token", "type", "username", "password", "config", "modules"}).
			AddRow(uint(1), time.Now(), time.Now(), "local", "http://localhost:2019", "", "local", "", "", "legacy-config", "{}"),
	)
	expectConfigWrite(mock, ConfigKey)

	for i := 0; i < 3; i++ {
		expectHasTable(mock, true)
		mock.ExpectExec(`UPDATE .* SET .*server_id`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	expectConfigWrite(mock, ConfigMigratedKey)

	MigrateLegacyConfig(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("单节点迁移未按预期执行: %v", err)
	}
}

// 老库多节点：只搬 id 最小的配置，其余不删数据（只在日志里告警）。
func TestMigrateLegacyConfigKeepsLowestIDOnly(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(configQueryRows())
	expectHasTable(mock, true)
	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "url", "token", "type", "username", "password", "config", "modules"}).
			AddRow(uint(1), time.Now(), time.Now(), "local", "http://localhost:2019", "", "local", "", "", "first-config", "{}").
			AddRow(uint(2), time.Now(), time.Now(), "remote", "http://10.0.0.2:2019", "", "remote", "", "", "second-config", "{}"),
	)
	expectConfigWrite(mock, ConfigKey)

	for i := 0; i < 3; i++ {
		expectHasTable(mock, true)
		mock.ExpectExec(`UPDATE .* SET .*server_id`).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	expectConfigWrite(mock, ConfigMigratedKey)

	MigrateLegacyConfig(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("多节点迁移未按预期执行: %v", err)
	}
}

// 表不存在时跳过：只跑迁移不应因缺表报错。
func TestMigrateLegacyConfigSkipsMissingTables(t *testing.T) {
	db, mock, cleanup := newConfigStoreMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).WillReturnRows(configQueryRows())
	expectHasTable(mock, false)
	for i := 0; i < 3; i++ {
		expectHasTable(mock, false)
	}
	expectConfigWrite(mock, ConfigMigratedKey)

	MigrateLegacyConfig(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("缺表时不应有任何写入: %v", err)
	}
}
