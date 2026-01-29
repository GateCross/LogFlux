package middleware

import (
	"encoding/json"
	"net/http"

	"logflux/common/result"
)

func ResponseMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default to 200
		}
		next(rw, r)

		// If status code is 200 and body is not empty, try to wrap it
		if rw.statusCode == http.StatusOK && len(rw.body) > 0 {
			var data interface{}
			// Try to unmarshal the existing body to see if it's JSON
			if err := json.Unmarshal(rw.body, &data); err == nil {
				// Optimization: Check if it's already wrapped to avoid double wrapping
				// We unmarshal into a map to check keys
				var tempMap map[string]interface{}
				if err := json.Unmarshal(rw.body, &tempMap); err == nil {
					// Check for "code" and "msg" keys to identify if it's likely already a ResponseBean
					if _, ok := tempMap["code"]; ok {
						if _, ok := tempMap["msg"]; ok {
							// Likely already wrapped (e.g. by HttpResult or logic), invalid to wrap again.
							// Just write original body.
							w.WriteHeader(rw.statusCode)
							w.Write(rw.body)
							return
						}
					}
				}

				// We assume standard goctl handlers return pure DTOs (data).
				// We wrap them into Result.ResponseBean.

				resp := result.ResponseBean{
					Code: 200,
					Msg:  "success",
					Data: data,
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			} else {
				// Not valid JSON, write original body
				w.WriteHeader(rw.statusCode)
				w.Write(rw.body)
			}
		} else {
			// For non-200 status or empty body, just write what we captured
			w.WriteHeader(rw.statusCode)
			w.Write(rw.body)
		}
	}
}

// Global Response Interceptor
func GlobalResponseHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           []byte{},
		}
		next.ServeHTTP(rw, r)

		// If status code is 200, we try to wrap it
		if rw.statusCode == http.StatusOK && len(rw.body) > 0 {
			// Check if it's already a ResponseBean structure?
			// Simpler: assume all 200 OK JSON logic responses need wrapping if they are not already.
			// However, raw bytes might be just a JSON string.

			// Let's rely on type assertion in logic if possible? No, middleware sees bytes.
			// Strategy: Unmarshal into interface{}, then wrap.

			var data interface{}
			if err := json.Unmarshal(rw.body, &data); err == nil {
				// Check if it already looks like {code, msg, data}?
				// This is tricky if APIs return similar structure.
				// But we know standard goctl handler returns pure DTO.

				// Double check if it's already wrapped (e.g. by SetErrorHandler for 200 errors)
				// If SetErrorHandler returns 200, it writes ResponseBean.
				// We can check if "code", "msg", "data" keys exist?
				// Safe bet: The user wants standard goctl handlers to work.
				// Standard goctl `httpx.OkJsonCtx` writes pure DTO.

				resp := result.ResponseBean{
					Code: 200,
					Msg:  "success",
					Data: data,
				}

				// Reset header if needed?
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			} else {
				// Not JSON or error, write original
				w.WriteHeader(rw.statusCode)
				w.Write(rw.body)
			}
		} else {
			// Non-200 or empty, write as is (Error handler usually handles errors)
			w.WriteHeader(rw.statusCode)
			w.Write(rw.body)
		}
	})
}

// Wrapper to capture status and body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// Don't write header yet, we might change it or body
}

func (w *responseWriter) Write(body []byte) (int, error) {
	w.body = append(w.body, body...)
	return len(body), nil
}
