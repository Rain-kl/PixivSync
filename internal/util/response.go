// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package util

// Response 通用响应体
type Response[T any] struct {
	ErrorMsg string `json:"error_msg"`
	Data     T      `json:"data"`
}

// ResponseAny 用于 Swagger 文档的响应类型（非泛型）
// swag 不支持泛型，使用此类型替代 Response[T]
type ResponseAny struct {
	ErrorMsg string      `json:"error_msg" example:""`
	Data     interface{} `json:"data"`
}

// OK 构造成功响应
func OK[T any](data T) Response[T] {
	return Response[T]{Data: data}
}

// OKNil 构造成功响应（data 为 null）
func OKNil() Response[any] {
	return Response[any]{Data: nil}
}

// Err 构造错误响应
func Err(msg string) Response[any] {
	return Response[any]{ErrorMsg: msg, Data: nil}
}
