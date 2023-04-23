package xrpc

import "fmt"

const (
	// Success 0 .
	Success = 0
	// ParseErr -32700 语法解析错误,服务端接收到无效的json。该错误发送于服务器尝试解析json文本
	ParseErr = -32700
	// InvalidRequest -32600 无效请求发送的json不是一个有效的请求对象。
	InvalidRequest = -32600
	// MethodNotFound -32601 找不到方法 该方法不存在或无效
	MethodNotFound = -32601
	// InvalidParamErr -32602 无效的参数 无效的方法参数。
	InvalidParamErr = -32602
	// InternalErr -32603 内部错误 JSON-RPC内部错误。
	InternalErr = -32603
	// ServerErr       = -32000 // ServerErr -32000 to -32099 Server error服务端错误, 预留用于自定义的服务器错误。
)

type Error struct {
	ErrCode int    `json:"code"`
	ErrMsg  string `json:"message"`
}

func (r *Error) Error() string {
	return fmt.Sprintf("Error(code: %d, errmsg: %s)", r.ErrCode, r.ErrMsg)
}

var errCodeMap = map[int]*Error{
	ParseErr:        &Error{ParseErr, "ParseErr"},
	InvalidRequest:  &Error{InvalidRequest, "InvalidRequest"},
	MethodNotFound:  &Error{MethodNotFound, "MethodNotFound"},
	InvalidParamErr: &Error{InvalidParamErr, "InvalidParamErr"},
	InternalErr:     &Error{InternalErr, "InternalErr"},
}
