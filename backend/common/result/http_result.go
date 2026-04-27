package result

import (
	"net/http"

	"logflux/internal/response"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Response Bean
type ResponseBean struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewErrMsg 返回自定义错误消息
func NewErrMsg(msg string) error {
	return xerr.NewBusinessErrorWith(msg)
}

// NewCodeError 返回自定义代码和消息的错误
func NewCodeError(code int, msg string) error {
	return xerr.NewCodeError(code, msg)
}

type CodeError = xerr.CodeError

// HttpResult
func HttpResult(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		httpx.OkJsonCtx(r.Context(), w, response.Success(resp))
	} else {
		logger.Errorc(r.Context(), err)
		httpx.ErrorCtx(r.Context(), w, err)
	}
}
