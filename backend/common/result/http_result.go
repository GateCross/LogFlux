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
		// 这里可以根据自定义错误类型进一步处理 Code
		errCode := 500
		errMsg := "服务器内部错误"

		// 简单的错误处理示例，实际项目中建议封装 errorx 包
		errMsg = err.Error()

		httpx.OkJson(w, ResponseBean{
			Code: errCode,
			Msg:  errMsg,
			Data: nil,
		})
	}
}
