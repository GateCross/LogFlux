package result

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Response Bean
type ResponseBean struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// NewErrMsg 返回自定义错误消息
func NewErrMsg(msg string) error {
	return &CodeError{Code: 400, Msg: msg}
}

// NewCodeError 返回自定义代码和消息的错误
func NewCodeError(code int, msg string) error {
	return &CodeError{Code: code, Msg: msg}
}

type CodeError struct {
	Code int
	Msg  string
}

func (e *CodeError) Error() string {
	return e.Msg
}

// HttpResult
func HttpResult(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		// 成功返回
		r := ResponseBean{
			Code: 200,
			Msg:  "success",
			Data: resp,
		}
		httpx.OkJson(w, r)
	} else {
		// 错误返回
		errCode := 500
		errMsg := err.Error()

		if ce, ok := err.(*CodeError); ok {
			errCode = ce.Code
			errMsg = ce.Msg
		}

		httpx.OkJson(w, ResponseBean{
			Code: errCode,
			Msg:  errMsg,
			Data: nil,
		})
	}
}
