package caddy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"
)

func mustCaddyTestServer(url string) *caddymodel.CaddyServer {
	return &caddymodel.CaddyServer{
		ID:   1,
		Name: "test",
		Url:  url,
		Type: "local",
	}
}

// TestCaddyConfigApplyRollsBackRuntimeWhenPersistFails 锁定单一写入器的失败语义：
// 落库失败时运行态必须加载回旧配置，而不是停在错误状态。
func TestCaddyConfigApplyRollsBackRuntimeWhenPersistFails(t *testing.T) {
	var mu sync.Mutex
	var posts []string
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if r.Method == http.MethodPost {
			mu.Lock()
			posts = append(posts, r.URL.Path+"\n"+string(body))
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newCaddySqlmockDB(t)
	defer cleanup()

	// 锁内读库内最新配置（system_configs）与各类单例模板
	expectLoadCurrentConfig(mock, integrationBaseConfig)
	expectMergeManagedBlocks(mock)
	mock.ExpectBegin()
	// SaveConfig 走 FirstOrCreate：键不存在时才 INSERT，这里让 INSERT 失败
	mock.ExpectQuery(`SELECT \* FROM "system_configs" WHERE .*key.* = \$1`).
		WithArgs(caddymodel.ConfigKey, 1).
		WillReturnRows(emptySystemConfigRows())
	mock.ExpectQuery(`INSERT INTO "system_configs"`).WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	srv := mustCaddyTestServer(caddyMock.URL)
	writer := newCaddyConfigApplyService(&svc.ServiceContext{DB: gdb}, nil)
	nextConfig := integrationBaseConfig + "\n# candidate\n"
	err := writer.Apply(srv, nextConfig, emptyModulesJSON, configWriteOptions{
		HistoryAction: "update",
	})
	if err == nil || !strings.Contains(err.Error(), "回滚") {
		t.Fatalf("expected persist failure to mention rollback, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	var loads []string
	for _, post := range posts {
		if strings.HasPrefix(post, "/load\n") {
			loads = append(loads, strings.TrimPrefix(post, "/load\n"))
		}
	}
	if len(loads) != 2 {
		t.Fatalf("expected candidate load + rollback load, got %d loads", len(loads))
	}
	if !strings.Contains(loads[0], "# candidate") {
		t.Fatalf("expected first load to carry candidate config, got %q", loads[0])
	}
	if strings.Contains(loads[1], "# candidate") {
		t.Fatalf("expected rollback load to restore previous config, got %q", loads[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

// TestCaddyConfigApplyRejectsStaleBaseHash 锁定乐观并发：哈希不一致时在 adapt 之前拒绝。
func TestCaddyConfigApplyRejectsStaleBaseHash(t *testing.T) {
	var requests int
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newCaddySqlmockDB(t)
	defer cleanup()

	expectLoadCurrentConfig(mock, integrationBaseConfig)

	srv := mustCaddyTestServer(caddyMock.URL)
	writer := newCaddyConfigApplyService(&svc.ServiceContext{DB: gdb}, nil)
	err := writer.Apply(srv, integrationBaseConfig, emptyModulesJSON, configWriteOptions{
		HistoryAction: "update",
		BaseHash:      "stale-hash",
	})
	if err == nil || !strings.Contains(err.Error(), "配置已被他人修改") {
		t.Fatalf("expected stale baseHash to be rejected, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("stale baseHash must be rejected before touching Caddy, got %d requests", requests)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

// TestCaddyConfigApplyRechecksBaseHashInsideLock 锁定乐观锁的兜底再校验语义：
// 请求开始时快照还干净，但等锁/等 adapt 期间他人已落库修改（锁内重读到脏配置），
// 必须以库内最新值为准拒绝，而不是拿"当时合法"的快照哈希放行。
func TestCaddyConfigApplyRechecksBaseHashInsideLock(t *testing.T) {
	var requests int
	caddyMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer caddyMock.Close()

	gdb, mock, cleanup := newCaddySqlmockDB(t)
	defer cleanup()

	// 锁内读到"他人已改过"的新配置，与请求快照不同
	snapshotConfig := integrationBaseConfig
	dirtyConfig := integrationBaseConfig + "\n# changed by someone else\n"
	expectLoadCurrentConfig(mock, dirtyConfig)

	srv := mustCaddyTestServer(caddyMock.URL)
	writer := newCaddyConfigApplyService(&svc.ServiceContext{DB: gdb}, nil)
	err := writer.Apply(srv, integrationBaseConfig, emptyModulesJSON, configWriteOptions{
		HistoryAction: "update",
		BaseHash:      hashConfig(snapshotConfig),
	})
	if err == nil || !strings.Contains(err.Error(), "配置已被他人修改") {
		t.Fatalf("expected dirty-in-window baseHash to be rejected, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("dirty baseHash must be rejected before touching Caddy, got %d requests", requests)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

// TestRollbackCaddyConfigGuardedByBaseHash 回滚路径同样受乐观并发保护：
// 校验由 Apply 锁内读库内最新配置统一执行，外层不再重复校验。
func TestRollbackCaddyConfigGuardedByBaseHash(t *testing.T) {
	gdb, mock, cleanup := newCaddySqlmockDB(t)
	defer cleanup()

	historyRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "server_id", "action", "hash", "config", "modules",
		}).AddRow(
			uint(7), time.Now(), time.Now(), uint(0), "update", "irrelevant",
			integrationBaseConfig, "{}",
		)
	}

	// logic 侧先取管理目标与历史记录
	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).
		WillReturnRows(caddyServerRows(time.Now(), "http://127.0.0.1:2019", ""))
	mock.ExpectQuery(`SELECT .* FROM "caddy_config_history"`).WillReturnRows(historyRows())
	// Apply 锁内读库内最新配置，baseHash 不一致在 adapt 之前拒绝
	expectLoadCurrentConfig(mock, integrationBaseConfig)

	logic := NewRollbackCaddyConfigLogic(context.Background(), &svc.ServiceContext{DB: gdb})
	_, err := logic.RollbackCaddyConfig(&types.CaddyConfigRollbackReq{
		HistoryId: 7,
		BaseHash:  "stale-hash",
	})
	if err == nil || !strings.Contains(err.Error(), "配置已被他人修改") {
		t.Fatalf("expected rollback with stale baseHash to be rejected, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
