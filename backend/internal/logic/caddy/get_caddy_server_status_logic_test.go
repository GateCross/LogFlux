package caddy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"logflux/internal/svc"
	"logflux/internal/types"
	caddymodel "logflux/model/caddy"
)

// Property 5: Probe 无副作用（Preservation 基线，Requirements 3.2/3.3/3.8）
// 任意探测请求不得调用 /load 或改写运行中配置；
// 单节点失败不得抹掉其他节点结果。
// **Validates: Requirements 3.2, 3.3, 3.8**（历史标注 4.2/4.3 为既有 property 编号）
// 与 TestProperty2_Preservation_StatusProbeObservableContracts 共同构成修复前后回归基线。

func TestProperty5_ProbeNoSideEffectsAndPartialSuccess(t *testing.T) {
	t.Run("empty nodes returns empty list not error", func(t *testing.T) {
		list := probeCaddyServers(context.Background(), nil, func(server *caddymodel.CaddyServer) error {
			t.Fatalf("probe must not be called when no servers")
			return nil
		})
		if list == nil {
			t.Fatalf("expected non-nil empty slice")
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d", len(list))
		}
	})

	t.Run("partial failure keeps successful results", func(t *testing.T) {
		servers := []caddymodel.CaddyServer{
			{ID: 1, Name: "ok-a", Url: "http://ok-a"},
			{ID: 2, Name: "bad", Url: "http://bad"},
			{ID: 3, Name: "ok-b", Url: "http://ok-b"},
		}
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			if server.ID == 2 {
				return fmt.Errorf("connection refused")
			}
			return nil
		})
		if len(list) != 3 {
			t.Fatalf("expected 3 results, got %d", len(list))
		}
		byID := map[uint]types.CaddyServerStatusItem{}
		for _, item := range list {
			byID[item.ServerId] = item
		}
		if !byID[1].Reachable || !byID[3].Reachable {
			t.Fatalf("successful nodes must remain reachable: %+v", list)
		}
		if byID[2].Reachable {
			t.Fatalf("failed node must be unreachable: %+v", byID[2])
		}
		if byID[2].ErrorMessage == "" {
			t.Fatalf("failed node must include Chinese error summary")
		}
		if !strings.Contains(byID[2].ErrorMessage, "连接") && !strings.Contains(byID[2].ErrorMessage, "探测") {
			t.Fatalf("expected Chinese error summary, got %q", byID[2].ErrorMessage)
		}
		for _, item := range list {
			if item.ProbedAt == "" {
				t.Fatalf("probedAt required for every item: %+v", item)
			}
			if item.Name == "" {
				t.Fatalf("name required: %+v", item)
			}
		}
	})

	t.Run("http probe only GET /config/ never /load", func(t *testing.T) {
		var loadHits atomic.Int64
		var configHits atomic.Int64
		var nonGetHits atomic.Int64
		var pathsMu sync.Mutex
		paths := make([]string, 0, 8)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			pathsMu.Lock()
			paths = append(paths, r.Method+" "+r.URL.Path)
			pathsMu.Unlock()
			if r.Method != http.MethodGet {
				nonGetHits.Add(1)
			}
			switch {
			case r.URL.Path == "/load" || strings.HasSuffix(r.URL.Path, "/load"):
				loadHits.Add(1)
				http.Error(w, "load must not be called by probe", http.StatusInternalServerError)
				return
			case r.URL.Path == "/config/" || r.URL.Path == "/config":
				configHits.Add(1)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
				return
			default:
				http.NotFound(w, r)
			}
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		servers := []caddymodel.CaddyServer{
			{ID: 10, Name: "node-a", Url: srv.URL, Token: "tok-a"},
			{ID: 11, Name: "node-b", Url: srv.URL, Token: "tok-b"},
		}
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			_, err := getCaddyConfigJSON(context.Background(), server)
			return err
		})
		if len(list) != 2 {
			t.Fatalf("expected 2 results, got %d", len(list))
		}
		for _, item := range list {
			if !item.Reachable {
				t.Fatalf("expected reachable, got %+v", item)
			}
		}
		if loadHits.Load() != 0 {
			t.Fatalf("probe must never call /load, hits=%d paths=%v", loadHits.Load(), paths)
		}
		if nonGetHits.Load() != 0 {
			t.Fatalf("probe must only use GET, non-GET hits=%d paths=%v", nonGetHits.Load(), paths)
		}
		if configHits.Load() < 2 {
			t.Fatalf("expected GET /config/ for each server, hits=%d paths=%v", configHits.Load(), paths)
		}
		for _, p := range paths {
			if strings.Contains(p, "/load") {
				t.Fatalf("forbidden path observed: %s", p)
			}
			if !strings.HasPrefix(p, "GET ") {
				t.Fatalf("non-GET probe traffic: %s", p)
			}
		}
	})

	// 多轮迭代：随机成败混合须始终返回全部节点，
	// 不产生 /load 副作用，且部分失败时保留成功结果。
	rng := rand.New(rand.NewSource(5))
	for i := 0; i < 100; i++ {
		n := 1 + rng.Intn(12)
		servers := make([]caddymodel.CaddyServer, n)
		failSet := make(map[uint]bool, n)
		for j := 0; j < n; j++ {
			id := uint(j + 1)
			servers[j] = caddymodel.CaddyServer{
				ID:   id,
				Name: fmt.Sprintf("s-%d", id),
				Url:  fmt.Sprintf("http://node-%d", id),
			}
			if rng.Intn(3) == 0 {
				failSet[id] = true
			}
		}
		// n >= 2 时保证至少一个失败与一个成功，覆盖部分成功属性
		if n >= 2 {
			failSet[1] = true
			failSet[2] = false
		}

		var loadLike atomic.Int64
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			// 仅模拟只读路径；不触碰 /load
			if strings.Contains(server.Url, "/load") {
				loadLike.Add(1)
			}
			if failSet[server.ID] {
				return fmt.Errorf("timeout waiting for response")
			}
			time.Sleep(time.Duration(rng.Intn(3)) * time.Millisecond)
			return nil
		})

		if loadLike.Load() != 0 {
			t.Fatalf("iter %d: probe must not target /load", i)
		}
		if len(list) != n {
			t.Fatalf("iter %d: expected %d results, got %d", i, n, len(list))
		}
		successCount := 0
		failCount := 0
		for _, item := range list {
			if item.ServerId == 0 || item.Name == "" || item.ProbedAt == "" {
				t.Fatalf("iter %d: incomplete item %+v", i, item)
			}
			wantFail := failSet[item.ServerId]
			if wantFail {
				failCount++
				if item.Reachable {
					t.Fatalf("iter %d: node %d should fail", i, item.ServerId)
				}
				if item.ErrorMessage == "" {
					t.Fatalf("iter %d: failed node missing errorMessage", i)
				}
			} else {
				successCount++
				if !item.Reachable {
					t.Fatalf("iter %d: node %d should succeed, err=%q", i, item.ServerId, item.ErrorMessage)
				}
				if item.ErrorMessage != "" {
					t.Fatalf("iter %d: success node must not carry errorMessage: %+v", i, item)
				}
			}
		}
		if n >= 2 && (successCount == 0 || failCount == 0) {
			t.Fatalf("iter %d: expected mixed outcomes, success=%d fail=%d", i, successCount, failCount)
		}
		// 属性：失败不得抹掉成功结果
		if successCount > 0 && failCount > 0 {
			for _, item := range list {
				if !failSet[item.ServerId] && !item.Reachable {
					t.Fatalf("iter %d: partial failure erased success for %d", i, item.ServerId)
				}
			}
		}
	}
}

func TestGetCaddyServerStatusLogic_UsesDBAndProbe(t *testing.T) {
	db, mock, cleanup := newStatusProbeMockDB(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"name", "url", "token", "type", "username", "password",
			"config", "modules",
		}).
			AddRow(uint(1), now, now, "live", "http://live", "", "local", "", "", "", "{}").
			AddRow(uint(2), now, now, "dead", "http://dead", "", "remote", "", "", "", "{}"),
	)

	logic := NewGetCaddyServerStatusLogic(context.Background(), &svc.ServiceContext{DB: db})
	logic.probeFn = func(server *caddymodel.CaddyServer) error {
		if server.ID == 2 {
			return fmt.Errorf("connection refused")
		}
		return nil
	}

	resp, err := logic.GetCaddyServerStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.List) != 2 {
		t.Fatalf("expected 2 status items, got %+v", resp)
	}
	if !resp.List[0].Reachable || resp.List[1].Reachable {
		// 顺序与 DB 查询顺序一致
		byID := map[uint]types.CaddyServerStatusItem{}
		for _, it := range resp.List {
			byID[it.ServerId] = it
		}
		if !byID[1].Reachable || byID[2].Reachable {
			t.Fatalf("unexpected reachability: %+v", resp.List)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGetCaddyServerStatusLogic_EmptyDB(t *testing.T) {
	db, mock, cleanup := newStatusProbeMockDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "created_at", "updated_at",
			"name", "url", "token", "type", "username", "password",
			"config", "modules",
		}),
	)

	logic := NewGetCaddyServerStatusLogic(context.Background(), &svc.ServiceContext{DB: db})
	resp, err := logic.GetCaddyServerStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.List == nil || len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGetCaddyConfigJSON_DoesNotHitLoad(t *testing.T) {
	// 直接单测：共享探测传输仅 GET /config/
	var sawLoad bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "load") {
			sawLoad = true
		}
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/config") {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, `{"apps":{}}`)
	}))
	defer srv.Close()

	body, err := getCaddyConfigJSON(context.Background(), &caddymodel.CaddyServer{Url: srv.URL, Token: "abc"})
	if err != nil {
		t.Fatalf("getCaddyConfigJSON: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("expected body")
	}
	if sawLoad {
		t.Fatalf("getCaddyConfigJSON must not call /load")
	}
}

func newStatusProbeMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
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
