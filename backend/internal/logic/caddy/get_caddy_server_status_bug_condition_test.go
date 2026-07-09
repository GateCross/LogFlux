package caddy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"logflux/internal/svc"
	"logflux/internal/xerr"
)

// Property 1: Bug Condition - Status Probe Safety and Correct Error/Context Handling
//
// 本探索测试编码「修复后」期望行为。在未修复代码上 MUST FAIL，以证明三类缺陷存在：
// 1) probeCaddyServers 裸 go（未 safego）
// 2) getCaddyConfigJSON 使用 http.NewRequest（无请求 context）
// 3) DB 查询失败返回裸 fmt.Errorf（非 xerr.CodeError）
//
// **Validates: Requirements 1.1, 1.2, 1.3, 2.1, 2.2, 2.3**

func TestProperty1_BugCondition_StatusProbeSafetyAndErrorContextHandling(t *testing.T) {
	t.Run("concurrent probe path uses safego not bare go", func(t *testing.T) {
		src := readCaddyLogicSource(t, "get_caddy_server_status_logic.go")
		fnBody := extractFunctionBody(t, src, "probeCaddyServers")

		// 期望：通过 safego.New(...).Go(...) 启动探测协程，并与 WaitGroup + 信号量配合
		if !strings.Contains(fnBody, "safego.New") {
			t.Fatalf("counterexample: probeCaddyServers 未使用 safego.New(...).Go(...) 启动探测协程（裸 go 路径）\nfunction body excerpt:\n%s", truncateForLog(fnBody, 800))
		}
		if !strings.Contains(fnBody, ".Go(") {
			t.Fatalf("counterexample: probeCaddyServers 缺少 safego.Go 调用\nfunction body excerpt:\n%s", truncateForLog(fnBody, 800))
		}
		// 外层不得再使用裸 go 启动探测（safego.Go 内部已 go）
		if hasBareGoLaunch(fnBody) {
			t.Fatalf("counterexample: probeCaddyServers 仍包含裸 go func 启动探测，未完全迁移到 safego\nfunction body excerpt:\n%s", truncateForLog(fnBody, 800))
		}
		// 保留有界并发编排契约
		if !strings.Contains(fnBody, "wg.Add") || !strings.Contains(fnBody, "wg.Wait") {
			t.Fatalf("expected WaitGroup coordination to remain in probeCaddyServers")
		}
		if !strings.Contains(fnBody, "probeConcurrency") && !strings.Contains(fnBody, "sem") {
			t.Fatalf("expected probeConcurrency semaphore coordination to remain in probeCaddyServers")
		}
	})

	t.Run("default probe HTTP uses NewRequestWithContext and request context", func(t *testing.T) {
		helpersSrc := readCaddyLogicSource(t, "caddy_helpers.go")
		helperBody := extractFunctionBody(t, helpersSrc, "getCaddyConfigJSON")

		// 期望：getCaddyConfigJSON 接收 context 并用 NewRequestWithContext
		if !strings.Contains(helperBody, "NewRequestWithContext") {
			t.Fatalf("counterexample: getCaddyConfigJSON 使用 http.NewRequest（无 context），未使用 http.NewRequestWithContext\nfunction body excerpt:\n%s", truncateForLog(helperBody, 600))
		}
		// 签名应包含 context.Context（修复后）
		sig := extractFunctionSignature(t, helpersSrc, "getCaddyConfigJSON")
		if !strings.Contains(sig, "context.Context") {
			t.Fatalf("counterexample: getCaddyConfigJSON 签名未接收 context.Context: %s", sig)
		}
		// 仍须只读 GET /config/，禁止 /load
		if !strings.Contains(helperBody, "/config/") {
			t.Fatalf("expected getCaddyConfigJSON to keep read-only GET /config/")
		}
		if strings.Contains(helperBody, "/load") {
			t.Fatalf("getCaddyConfigJSON must not call /load")
		}

		statusSrc := readCaddyLogicSource(t, "get_caddy_server_status_logic.go")
		// 默认 probe 闭包应把请求 ctx 传入 helper
		if !regexp.MustCompile(`getCaddyConfigJSON\s*\(\s*l\.ctx\s*,`).MatchString(statusSrc) &&
			!regexp.MustCompile(`getCaddyConfigJSON\s*\(\s*ctx\s*,`).MatchString(statusSrc) {
			t.Fatalf("counterexample: GetCaddyServerStatus 默认 probe 未将请求 context 传入 getCaddyConfigJSON（无 context HTTP 路径）")
		}
	})

	t.Run("DB list query failure returns xerr CodeError with Chinese message", func(t *testing.T) {
		db, mock, cleanup := newStatusProbeMockDB(t)
		defer cleanup()

		dbErr := errors.New("sql: connection refused")
		mock.ExpectQuery(`SELECT \* FROM "caddy_servers"`).WillReturnError(dbErr)

		logic := NewGetCaddyServerStatusLogic(context.Background(), &svc.ServiceContext{DB: db})
		resp, err := logic.GetCaddyServerStatus()
		if resp != nil {
			t.Fatalf("expected nil resp on DB failure, got %+v", resp)
		}
		if err == nil {
			t.Fatalf("expected error on DB failure")
		}

		// 期望：xerr.NewCodeErrorWithCause(ServerCommonError, "查询 Caddy 节点失败", err)
		var codeErr *xerr.CodeError
		if !errors.As(err, &codeErr) || codeErr == nil {
			t.Fatalf("counterexample: GetCaddyServerStatus DB 失败返回裸错误（非 *xerr.CodeError）: type=%T err=%v", err, err)
		}
		if codeErr.Code != xerr.ServerCommonError {
			t.Fatalf("expected ServerCommonError (%d), got %d", xerr.ServerCommonError, codeErr.Code)
		}
		if !strings.Contains(codeErr.Message, "查询 Caddy 节点失败") {
			t.Fatalf("expected Chinese message containing 查询 Caddy 节点失败, got %q", codeErr.Message)
		}
		if !errors.Is(err, dbErr) {
			// 根因应可 unwrap（NewCodeErrorWithCause）
			if cause := errors.Unwrap(err); cause == nil || !strings.Contains(cause.Error(), "connection refused") {
				t.Fatalf("expected wrapped DB cause to remain unwrap-able, err=%v", err)
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})
}

func readCaddyLogicSource(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func extractFunctionSignature(t *testing.T, src, funcName string) string {
	t.Helper()
	// 匹配: func name(...) 或 func (recv) name(...)
	re := regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s+)?` + regexp.QuoteMeta(funcName) + `\s*\([^)]*\)[^{]*`)
	m := re.FindString(src)
	if m == "" {
		t.Fatalf("function %s signature not found", funcName)
	}
	return strings.TrimSpace(m)
}

func extractFunctionBody(t *testing.T, src, funcName string) string {
	t.Helper()
	// 定位函数起始
	re := regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s+)?` + regexp.QuoteMeta(funcName) + `\s*\(`)
	loc := re.FindStringIndex(src)
	if loc == nil {
		t.Fatalf("function %s not found", funcName)
	}
	// 从签名后第一个 '{' 起做括号配对
	start := strings.Index(src[loc[0]:], "{")
	if start < 0 {
		t.Fatalf("function %s body '{' not found", funcName)
	}
	start = loc[0] + start
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1]
			}
		}
	}
	t.Fatalf("function %s body not closed", funcName)
	return ""
}

// hasBareGoLaunch 检测探测函数体内是否存在裸 go func / go 启动（非注释）。
func hasBareGoLaunch(fnBody string) bool {
	// 去掉行注释后匹配 "go func" 或 "go (" 形式的裸启动
	var cleaned strings.Builder
	for _, line := range strings.Split(fnBody, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		cleaned.WriteString(line)
		cleaned.WriteByte('\n')
	}
	body := cleaned.String()
	// 裸 go func(...) 或 go someFunc(
	if strings.Contains(body, "go func") {
		return true
	}
	// 避免匹配 safego 包名；匹配独立的 go 关键字启动
	re := regexp.MustCompile(`\bgo\s+[A-Za-z_(]`)
	return re.MatchString(body)
}

func truncateForLog(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated)"
}
