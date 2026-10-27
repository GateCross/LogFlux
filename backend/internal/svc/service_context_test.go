package svc

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	caddymodel "logflux/model/caddy"
)

// 播种默认节点的三个分支：标记键已存在、已有节点、首次空库。
//
// 三条用例都用 sqlmock 断言 SQL 序列本身，因为这里的行为差异完全体现在
// 「查不查 count」「插不插 caddy_servers」「写不写标记」这三件事上。

// 标记键已存在：不再统计数量、不插入任何行。
func TestEnsureDefaultCaddyServerSkipsWhenMarkerExists(t *testing.T) {
	db, mock, cleanup := newSvcMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE key = \$1 ORDER BY "system_configs"\."id" LIMIT \$2`).
		WithArgs(systemConfigKeyNodeSeeded, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "key", "value",
		}).AddRow(uint(1), nil, nil, systemConfigKeyNodeSeeded, "2026-01-01T00:00:00Z"))

	ensureDefaultCaddyServer(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("标记键存在时不应有其它 SQL: %v", err)
	}
}

// 标记键不存在但已有节点：只补标记，不插节点。
func TestEnsureDefaultCaddyServerWritesMarkerOnlyWhenNodesExist(t *testing.T) {
	db, mock, cleanup := newSvcMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE key = \$1 ORDER BY "system_configs"\."id" LIMIT \$2`).
		WithArgs(systemConfigKeyNodeSeeded, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "key", "value",
		}))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "caddy_servers"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	// FirstOrCreate 先按条件查一次，查不到才 INSERT。
	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE "system_configs"\."key" = \$1 ORDER BY "system_configs"\."id" LIMIT \$2`).
		WithArgs(systemConfigKeyNodeSeeded, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "key", "value",
		}))
	mock.ExpectQuery(`INSERT INTO "system_configs" .*RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(2)))

	ensureDefaultCaddyServer(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("已有节点时不应插入 caddy_servers: %v", err)
	}
}

// 空库：插入一条 local 节点（modules 必须是 "{}"），再写标记。
func TestEnsureDefaultCaddyServerSeedsLocalNodeWhenEmpty(t *testing.T) {
	db, mock, cleanup := newSvcMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE key = \$1 ORDER BY "system_configs"\."id" LIMIT \$2`).
		WithArgs(systemConfigKeyNodeSeeded, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "key", "value",
		}))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "caddy_servers"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))

	mock.ExpectQuery(`INSERT INTO "caddy_servers" .*RETURNING "id"`).
		WithArgs(
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			defaultCaddyServerName,
			defaultCaddyServerURL,
			"", // token
			"local",
			"",   // username
			"",   // password
			"",   // config
			"{}", // modules：jsonb 列不接受空串
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(1)))

	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE "system_configs"\."key" = \$1 ORDER BY "system_configs"\."id" LIMIT \$2`).
		WithArgs(systemConfigKeyNodeSeeded, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "key", "value",
		}))
	mock.ExpectQuery(`INSERT INTO "system_configs" .*RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(2)))

	ensureDefaultCaddyServer(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("空库应插入节点并写标记: %v", err)
	}
}

// 启动顺序：迁移必须先于播种执行，否则新库播种的 local 行会先被写出来，
// 老库则可能因为读不到 caddy_config 而在迁移前被当作「未接管」。
func TestStartupMigratesConfigBeforeSeeding(t *testing.T) {
	db, mock, cleanup := newSvcMockDB(t)
	defer cleanup()

	// 迁移：配置键不存在 → 读 caddy_servers 迁移 → 三张表归零 → 写标记
	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"}))
	mock.ExpectQuery(`SELECT count\(\*\) FROM information_schema\.tables`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "url", "token", "type", "username", "password", "config", "modules"}).
			AddRow(uint(1), nil, nil, "local", "http://localhost:2019", "", "local", "", "", "legacy", "{}"),
	)
	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"}))
	mock.ExpectQuery(`INSERT INTO "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(2)))
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`SELECT count\(\*\) FROM information_schema\.tables`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
		mock.ExpectExec(`UPDATE .* SET .*server_id`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"}))
	mock.ExpectQuery(`INSERT INTO "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)))

	// 播种：标记键已存在 → 不再查 count、不插节点
	mock.ExpectQuery(`SELECT \* FROM "system_configs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "key", "value"}).
			AddRow(uint(4), nil, nil, systemConfigKeyNodeSeeded, "2026-01-01T00:00:00Z"))

	caddymodel.MigrateLegacyConfig(db)
	ensureDefaultCaddyServer(db)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("启动迁移与播种顺序不符: %v", err)
	}
}

// newSvcMockDB 建一个 sqlmock 驱动的 GORM 句柄（postgres 方言）。
func newSvcMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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
