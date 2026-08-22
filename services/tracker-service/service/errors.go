package service

import "errors"

// 业务层错误定义。handler 层负责转换为 grpc/http 状态码。
var (
	// errBadAnnounce 表示 announce 请求参数非法。
	errBadAnnounce = errors.New("bad announce request")
)
