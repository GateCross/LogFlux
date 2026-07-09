package caddy

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"logflux/internal/types"
	caddymodel "logflux/model/caddy"
)

// Property 2: Preservation - Status Probe Observable Contracts
//
// Observation-first: encode existing observable contracts on UNFIXED code
// (orthogonal to the three defect paths). After the fix these tests MUST still pass.
//
// Baseline tests in the same package:
//   - TestProperty5_ProbeNoSideEffectsAndPartialSuccess
//   - TestGetCaddyServerStatusLogic_EmptyDB
//   - TestGetCaddyConfigJSON_DoesNotHitLoad
//
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8**

func TestProperty2_Preservation_StatusProbeObservableContracts(t *testing.T) {
	// 3.1 empty registered nodes -> non-nil empty List, no probe calls
	t.Run("empty registered nodes return non-nil empty list without probing", func(t *testing.T) {
		var probeCalls atomic.Int64
		list := probeCaddyServers(context.Background(), nil, func(server *caddymodel.CaddyServer) error {
			probeCalls.Add(1)
			t.Fatalf("probe must not be called for empty server list")
			return nil
		})
		if list == nil {
			t.Fatalf("expected non-nil empty slice, got nil")
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got len=%d", len(list))
		}
		if probeCalls.Load() != 0 {
			t.Fatalf("expected zero probe calls, got %d", probeCalls.Load())
		}

		list2 := probeCaddyServers(context.Background(), []caddymodel.CaddyServer{}, func(server *caddymodel.CaddyServer) error {
			t.Fatalf("probe must not be called for zero-length server slice")
			return nil
		})
		if list2 == nil || len(list2) != 0 {
			t.Fatalf("expected non-nil empty slice for zero-length input, got %+v", list2)
		}
	})

	// 3.2 + 3.5 partial success: 1:1 results; success has Reachable + LatencyMs + ProbedAt
	t.Run("multi-node mixed outcomes preserve 1:1 results with success fields", func(t *testing.T) {
		servers := []caddymodel.CaddyServer{
			{ID: 1, Name: "ok-a", Url: "http://ok-a"},
			{ID: 2, Name: "bad", Url: "http://bad"},
			{ID: 3, Name: "ok-b", Url: "http://ok-b"},
			{ID: 4, Name: "timeout", Url: "http://timeout"},
		}
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			switch server.ID {
			case 2:
				return fmt.Errorf("connection refused")
			case 4:
				return fmt.Errorf("timeout waiting for response")
			default:
				time.Sleep(2 * time.Millisecond)
				return nil
			}
		})
		if len(list) != len(servers) {
			t.Fatalf("partial success must return full 1:1 list: want %d got %d", len(servers), len(list))
		}
		byID := indexStatusByID(list)
		if !byID[1].Reachable || !byID[3].Reachable {
			t.Fatalf("successful nodes must remain reachable: %+v", list)
		}
		if byID[2].Reachable || byID[4].Reachable {
			t.Fatalf("failed nodes must be unreachable: %+v", list)
		}
		for _, id := range []uint{1, 3} {
			item := byID[id]
			if item.ErrorMessage != "" {
				t.Fatalf("success node %d must not carry ErrorMessage: %+v", id, item)
			}
			if item.ProbedAt == "" {
				t.Fatalf("success node %d missing ProbedAt: %+v", id, item)
			}
			if item.LatencyMs < 0 {
				t.Fatalf("success node %d LatencyMs must be non-negative: %+v", id, item)
			}
			if item.Name == "" || item.ServerId != id {
				t.Fatalf("success node identity broken: %+v", item)
			}
		}
		for _, id := range []uint{2, 4} {
			if byID[id].ErrorMessage == "" {
				t.Fatalf("failed node %d missing Chinese ErrorMessage", id)
			}
			if byID[id].ProbedAt == "" {
				t.Fatalf("failed node %d missing ProbedAt", id)
			}
		}
	})

	// 3.3 probe transport is read-only GET /config/ only (never /load)
	t.Run("probe transport is read-only GET /config/ only", func(t *testing.T) {
		var loadHits atomic.Int64
		var configHits atomic.Int64
		var nonGetHits atomic.Int64
		var pathsMu sync.Mutex
		var paths []string

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
			{ID: 20, Name: "node-a", Url: srv.URL, Token: "tok-a"},
			{ID: 21, Name: "node-b", Url: srv.URL + "/", Token: "tok-b"},
		}
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			// Matches Status default path: shared read-only helper (ctx + GET /config/)
			_, err := getCaddyConfigJSON(context.Background(), server)
			return err
		})
		if len(list) != 2 {
			t.Fatalf("expected 2 results, got %d", len(list))
		}
		for _, item := range list {
			if !item.Reachable {
				t.Fatalf("expected reachable via GET /config/, got %+v", item)
			}
			if item.LatencyMs < 0 || item.ProbedAt == "" {
				t.Fatalf("success fields incomplete: %+v", item)
			}
		}
		if loadHits.Load() != 0 {
			t.Fatalf("probe must never call /load, hits=%d paths=%v", loadHits.Load(), paths)
		}
		if nonGetHits.Load() != 0 {
			t.Fatalf("probe must only use GET, non-GET hits=%d paths=%v", nonGetHits.Load(), paths)
		}
		if configHits.Load() < 2 {
			t.Fatalf("expected GET /config/ per server, hits=%d paths=%v", configHits.Load(), paths)
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

	// 3.4 failures keep short Chinese ErrorMessage summaries (categories + truncate)
	t.Run("probe failures produce short Chinese ErrorMessage summaries", func(t *testing.T) {
		cases := []struct {
			name       string
			err        error
			wantSubstr []string
		}{
			{
				name:       "timeout",
				err:        fmt.Errorf("Client.Timeout exceeded while awaiting headers"),
				wantSubstr: []string{"探测超时"},
			},
			{
				name:       "deadline",
				err:        fmt.Errorf("context deadline exceeded"),
				wantSubstr: []string{"探测超时"},
			},
			{
				name:       "connection refused",
				err:        fmt.Errorf("dial tcp 127.0.0.1:2019: connection refused"),
				wantSubstr: []string{"连接被拒绝"},
			},
			{
				name:       "host resolve",
				err:        fmt.Errorf("lookup caddy.invalid: no such host"),
				wantSubstr: []string{"主机解析失败"},
			},
			{
				name:       "auth 401",
				err:        fmt.Errorf("unauthorized: 401"),
				wantSubstr: []string{"鉴权失败"},
			},
			{
				name:       "auth 403",
				err:        fmt.Errorf("forbidden 403"),
				wantSubstr: []string{"鉴权失败"},
			},
			{
				name:       "already chinese config error",
				err:        fmt.Errorf("获取 Caddy 配置失败: bad gateway"),
				wantSubstr: []string{"获取 Caddy 配置失败"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				msg := summarizeProbeError(tc.err)
				if msg == "" {
					t.Fatalf("expected non-empty Chinese summary for %v", tc.err)
				}
				if !containsNonASCII(msg) {
					t.Fatalf("expected Chinese summary, got %q", msg)
				}
				for _, sub := range tc.wantSubstr {
					if !strings.Contains(msg, sub) {
						t.Fatalf("summary %q missing %q", msg, sub)
					}
				}
			})
		}

		longBody := strings.Repeat("x", 500)
		shortMsg := truncateProbeMessage("探测失败: " + longBody)
		if !strings.HasSuffix(shortMsg, "...") {
			t.Fatalf("expected truncation suffix, got %q", shortMsg)
		}
		if len([]rune(shortMsg)) > 203 {
			t.Fatalf("truncated message too long: runes=%d", len([]rune(shortMsg)))
		}

		item := probeOneServer(&caddymodel.CaddyServer{ID: 9, Name: "n"}, func(server *caddymodel.CaddyServer) error {
			return fmt.Errorf("connection refused")
		})
		if item.Reachable || item.ErrorMessage == "" {
			t.Fatalf("probeOneServer failure shape broken: %+v", item)
		}
		if !strings.Contains(item.ErrorMessage, "连接") {
			t.Fatalf("expected Chinese connection summary, got %q", item.ErrorMessage)
		}
	})

	// 3.5 success marks Reachable=true with LatencyMs and ProbedAt
	t.Run("probe success marks Reachable with LatencyMs and ProbedAt", func(t *testing.T) {
		item := probeOneServer(&caddymodel.CaddyServer{ID: 7, Name: "ok"}, func(server *caddymodel.CaddyServer) error {
			time.Sleep(3 * time.Millisecond)
			return nil
		})
		if !item.Reachable {
			t.Fatalf("expected Reachable=true, got %+v", item)
		}
		if item.ErrorMessage != "" {
			t.Fatalf("success must not set ErrorMessage: %+v", item)
		}
		if item.ProbedAt == "" {
			t.Fatalf("success missing ProbedAt: %+v", item)
		}
		if item.LatencyMs < 0 {
			t.Fatalf("LatencyMs must be non-negative: %+v", item)
		}
		if item.ServerId != 7 || item.Name != "ok" {
			t.Fatalf("identity fields broken: %+v", item)
		}
	})

	// 3.6 node count > probeConcurrency remains semaphore-bounded
	t.Run("node count above probeConcurrency remains semaphore-bounded", func(t *testing.T) {
		const n = probeConcurrency + 8
		servers := make([]caddymodel.CaddyServer, n)
		for i := 0; i < n; i++ {
			servers[i] = caddymodel.CaddyServer{
				ID:   uint(i + 1),
				Name: fmt.Sprintf("s-%d", i+1),
				Url:  fmt.Sprintf("http://node-%d", i+1),
			}
		}

		var (
			inFlight    atomic.Int64
			maxInFlight atomic.Int64
		)
		list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
			cur := inFlight.Add(1)
			for {
				prev := maxInFlight.Load()
				if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(15 * time.Millisecond)
			inFlight.Add(-1)
			return nil
		})

		if len(list) != n {
			t.Fatalf("expected %d results under bounded concurrency, got %d", n, len(list))
		}
		peak := maxInFlight.Load()
		if peak > int64(probeConcurrency) {
			t.Fatalf("concurrency peak %d exceeds probeConcurrency=%d", peak, probeConcurrency)
		}
		if peak < 1 {
			t.Fatalf("expected concurrent probes, peak=%d", peak)
		}
		for _, item := range list {
			if !item.Reachable || item.ProbedAt == "" {
				t.Fatalf("bounded-concurrency success item incomplete: %+v", item)
			}
		}
	})

	// 3.7 getCaddyConfigJSON outside Status keeps read-only success/failure semantics
	t.Run("getCaddyConfigJSON outside Status keeps read-only success and failure semantics", func(t *testing.T) {
		var sawLoad atomic.Bool
		okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "load") {
				sawLoad.Store(true)
			}
			if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/config") {
				http.Error(w, "unexpected", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"apps":{}}`))
		}))
		defer okSrv.Close()

		body, err := getCaddyConfigJSON(context.Background(), &caddymodel.CaddyServer{Url: okSrv.URL, Token: "t"})
		if err != nil {
			t.Fatalf("success path: %v", err)
		}
		if len(body) == 0 {
			t.Fatalf("expected config body")
		}
		if sawLoad.Load() {
			t.Fatalf("getCaddyConfigJSON must not call /load")
		}

		failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		}))
		defer failSrv.Close()
		_, err = getCaddyConfigJSON(context.Background(), &caddymodel.CaddyServer{Url: failSrv.URL})
		if err == nil {
			t.Fatalf("expected failure on non-2xx /config/")
		}
		if !strings.Contains(err.Error(), "获取 Caddy 配置失败") {
			t.Fatalf("expected Chinese config failure message, got %v", err)
		}
	})

	// 3.2 / 3.8 property: random reachability combinations preserve full result list
	t.Run("property: random reachability combinations preserve full result list", func(t *testing.T) {
		rng := rand.New(rand.NewSource(42))
		for i := 0; i < 80; i++ {
			n := 1 + rng.Intn(14)
			servers := make([]caddymodel.CaddyServer, n)
			failSet := make(map[uint]bool, n)
			for j := 0; j < n; j++ {
				id := uint(j + 1)
				servers[j] = caddymodel.CaddyServer{
					ID:   id,
					Name: fmt.Sprintf("s-%d", id),
					Url:  fmt.Sprintf("http://node-%d", id),
				}
				if rng.Intn(2) == 0 {
					failSet[id] = true
				}
			}
			// Force mixed outcomes when n >= 2 for partial-success coverage
			if n >= 2 {
				failSet[1] = true
				failSet[2] = false
			}

			list := probeCaddyServers(context.Background(), servers, func(server *caddymodel.CaddyServer) error {
				if failSet[server.ID] {
					switch server.ID % 4 {
					case 0:
						return fmt.Errorf("timeout waiting")
					case 1:
						return fmt.Errorf("connection refused")
					case 2:
						return fmt.Errorf("no such host")
					default:
						return fmt.Errorf("unauthorized 401")
					}
				}
				if rng.Intn(4) == 0 {
					time.Sleep(time.Millisecond)
				}
				return nil
			})

			if len(list) != n {
				t.Fatalf("iter %d: expected %d results, got %d", i, n, len(list))
			}
			successCount, failCount := 0, 0
			for idx, item := range list {
				// Results written by index; order matches input
				wantID := servers[idx].ID
				if item.ServerId != wantID {
					t.Fatalf("iter %d: index %d want ServerId=%d got %d", i, idx, wantID, item.ServerId)
				}
				if item.Name == "" || item.ProbedAt == "" {
					t.Fatalf("iter %d: incomplete item %+v", i, item)
				}
				if item.LatencyMs < 0 {
					t.Fatalf("iter %d: negative LatencyMs %+v", i, item)
				}
				wantFail := failSet[item.ServerId]
				if wantFail {
					failCount++
					if item.Reachable {
						t.Fatalf("iter %d: node %d should fail", i, item.ServerId)
					}
					if item.ErrorMessage == "" || !containsNonASCII(item.ErrorMessage) {
						t.Fatalf("iter %d: failed node missing Chinese ErrorMessage: %+v", i, item)
					}
				} else {
					successCount++
					if !item.Reachable {
						t.Fatalf("iter %d: node %d should succeed err=%q", i, item.ServerId, item.ErrorMessage)
					}
					if item.ErrorMessage != "" {
						t.Fatalf("iter %d: success must not carry ErrorMessage: %+v", i, item)
					}
				}
			}
			if n >= 2 && (successCount == 0 || failCount == 0) {
				t.Fatalf("iter %d: expected mixed outcomes success=%d fail=%d", i, successCount, failCount)
			}
			if successCount > 0 && failCount > 0 {
				for _, item := range list {
					if !failSet[item.ServerId] && !item.Reachable {
						t.Fatalf("iter %d: partial failure erased success for %d", i, item.ServerId)
					}
				}
			}
		}
	})
}

func indexStatusByID(list []types.CaddyServerStatusItem) map[uint]types.CaddyServerStatusItem {
	out := make(map[uint]types.CaddyServerStatusItem, len(list))
	for _, item := range list {
		out[item.ServerId] = item
	}
	return out
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}
