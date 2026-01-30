package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"logflux/common/result"

	"github.com/zeromicro/go-zero/core/logx"
)

// responseWriter 池，复用对象减少 GC 压力
var rwPool = sync.Pool{
	New: func() interface{} {
		return &responseWriter{
			body: bytes.NewBuffer(make([]byte, 0, 1024)),
		}
	},
}

// ResponseMiddleware 全局响应中间件
// 自动将未包装的 JSON 响应包装为 {code, msg, data} 格式
func ResponseMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从池中获取 responseWriter
		rw := rwPool.Get().(*responseWriter)
		rw.ResponseWriter = w
		rw.statusCode = http.StatusOK
		rw.body.Reset()

		defer func() {
			// 归还到池中
			rwPool.Put(rw)
		}()

		next(rw, r)

		bodyBytes := rw.body.Bytes()

		// 非 200 或空 body，直接写入原始响应
		if rw.statusCode != http.StatusOK || len(bodyBytes) == 0 {
			w.WriteHeader(rw.statusCode)
			w.Write(bodyBytes)
			return
		}

		// 单次 JSON 解析：直接解析为 map 检查是否已包装
		var dataMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &dataMap); err != nil {
			// 非 JSON 响应，直接写入
			logx.Debugf("响应非 JSON 格式，跳过包装: %s", r.URL.Path)
			w.WriteHeader(rw.statusCode)
			w.Write(bodyBytes)
			return
		}

		// 检查是否已包装（包含 code 和 msg 字段）
		_, hasCode := dataMap["code"]
		_, hasMsg := dataMap["msg"]
		if hasCode && hasMsg {
			// 已包装，直接写入
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(bodyBytes)
			return
		}

		// 未包装，进行包装
		resp := result.ResponseBean{
			Code: 200,
			Msg:  "success",
			Data: dataMap,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logx.Errorf("响应编码失败: %v", err)
		}
	}
}

// responseWriter 包装器，捕获状态码和响应体
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// 延迟写入 header，可能需要修改
}

func (w *responseWriter) Write(body []byte) (int, error) {
	return w.body.Write(body)
}
