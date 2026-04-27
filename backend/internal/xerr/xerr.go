package xerr

import "errors"

const (
	// OK 表示业务处理成功。
	OK = 0

	// BusinessCommonError 表示通用业务校验错误。
	BusinessCommonError = 400
	// Unauthorized 表示登录态无效或缺失。
	Unauthorized = 401
	// Forbidden 表示当前用户没有操作权限。
	Forbidden = 403
	// NotFound 表示业务资源不存在。
	NotFound = 404
	// ServerCommonError 表示需要展示给前端的系统错误。
	ServerCommonError = 500
)

var errMsg = map[int]string{
	OK:                  "成功",
	BusinessCommonError: "请求参数或业务规则不满足",
	Unauthorized:        "登录状态无效",
	Forbidden:           "权限不足",
	NotFound:            "资源不存在",
	ServerCommonError:   "系统繁忙，请稍后重试",
}

// CodeError 是全局业务错误类型，供 httpx.SetErrorHandler 统一转换响应。
type CodeError struct {
	Code    int
	Message string
	cause   error
}

func (e *CodeError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *CodeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// NewCodeError 返回指定错误码和中文错误信息。
func NewCodeError(code int, message string) error {
	if message == "" {
		message = MapErrMsg(code)
	}
	return &CodeError{Code: code, Message: message}
}

// NewCodeErrorWithCause 返回带根因的错误，根因只用于日志和 errors.Is/As。
func NewCodeErrorWithCause(code int, message string, cause error) error {
	if message == "" {
		message = MapErrMsg(code)
	}
	return &CodeError{Code: code, Message: message, cause: cause}
}

// NewBusinessErrorWith 返回业务校验错误。
func NewBusinessErrorWith(message string) error {
	return NewCodeError(BusinessCommonError, message)
}

// NewSystemErrorWith 返回需要展示给前端的系统错误。
func NewSystemErrorWith(message string) error {
	return NewCodeError(ServerCommonError, message)
}

// NewEnumError 根据错误码返回枚举错误。
func NewEnumError(code int) error {
	return NewCodeError(code, MapErrMsg(code))
}

// MapErrMsg 返回错误码对应的中文文案。
func MapErrMsg(code int) string {
	if msg, ok := errMsg[code]; ok {
		return msg
	}
	return errMsg[ServerCommonError]
}

// CodeFromError 从错误中提取业务错误码，未知错误返回系统错误码。
func CodeFromError(err error) int {
	var codeErr *CodeError
	if errors.As(err, &codeErr) && codeErr != nil {
		return codeErr.Code
	}
	return ServerCommonError
}

// MessageFromError 从错误中提取前端展示文案。
func MessageFromError(err error) string {
	if err == nil {
		return MapErrMsg(OK)
	}
	var codeErr *CodeError
	if errors.As(err, &codeErr) && codeErr != nil {
		return codeErr.Message
	}
	return err.Error()
}
