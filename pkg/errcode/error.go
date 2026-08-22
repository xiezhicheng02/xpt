// Package errcode 定义跨服务共享的业务错误码。
//
// 错误码分两类：
//   - 业务错误：以 Err 开头，供 service/handler 层判断分支；
//   - 传输错误：由 errcode 的 grpc/http 映射函数转换为具体状态。
//
// 学习提示：private tracker 场景常见的失败原因都列在这里，
// 实现各服务时直接复用，避免散落魔法字符串。
package errcode

import (
	"errors"
	"fmt"
)

// Code 是稳定的业务错误码，跨服务传输时保持数值不变。
type Code int

const (
	// CodeOK 无错误。
	CodeOK Code = 0

	// CodeInternal 内部错误（数据库、未知异常）。
	CodeInternal Code = 1
	// CodeInvalidArgument 参数校验失败。
	CodeInvalidArgument Code = 2
	// CodeNotFound 资源不存在。
	CodeNotFound Code = 3
	// CodeUnauthorized 未认证（token 缺失/无效）。
	CodeUnauthorized Code = 4
	// CodeForbidden 已认证但无权限。
	CodeForbidden Code = 5
	// CodeConflict 资源冲突（重复注册、重复上传等）。
	CodeConflict Code = 6
)

// Error 是带错误码的业务错误。
type Error struct {
	Code Code
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("errcode=%d msg=%s", e.Code, e.Msg)
}

// New 构造一个带错误码的错误。
func New(code Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

// Wrap 包装底层错误并附加错误码。
func Wrap(code Code, err error) *Error {
	return &Error{Code: code, Msg: err.Error()}
}

// FromError 从任意 error 提取 *Error；非业务错误统一映射为 CodeInternal。
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: CodeInternal, Msg: err.Error()}
}

// ---- 常用业务错误 ----

// 认证与用户。
var (
	ErrUnauthorized = New(CodeUnauthorized, "unauthorized")
	ErrForbidden    = New(CodeForbidden, "forbidden")
	ErrUserExists   = New(CodeConflict, "user already exists")
	ErrBadLogin     = New(CodeUnauthorized, "bad username or password")
)

// 种子与 announce。
var (
	ErrTorrentNotFound = New(CodeNotFound, "torrent not found")
	ErrBadAnnounce     = New(CodeInvalidArgument, "bad announce request")
	ErrBadInfoHash     = New(CodeInvalidArgument, "bad info hash")
	ErrPeerNotFound    = New(CodeNotFound, "peer not found")
)

// 通用。
var (
	ErrBadRequest = New(CodeInvalidArgument, "bad request")
	ErrNotFound   = New(CodeNotFound, "not found")
	ErrInternal   = New(CodeInternal, "internal error")
)
