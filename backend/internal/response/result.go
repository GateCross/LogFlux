package response

import "logflux/internal/xerr"

// Result 是所有 HTTP JSON 接口统一返回结构。
// Msg 仅用于兼容旧前端，业务代码应使用 Message。
type Result struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Msg     string      `json:"msg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 返回成功响应。
func Success(data interface{}) Result {
	message := xerr.MapErrMsg(xerr.OK)
	return Result{
		Code:    xerr.OK,
		Message: message,
		Msg:     message,
		Data:    data,
	}
}

// Error 返回错误响应。
func Error(code int, message string) Result {
	if message == "" {
		message = xerr.MapErrMsg(code)
	}
	return Result{
		Code:    code,
		Message: message,
		Msg:     message,
	}
}

// ErrorFromErr 将 error 转为统一错误响应。
func ErrorFromErr(err error) Result {
	return Error(xerr.CodeFromError(err), xerr.MessageFromError(err))
}
