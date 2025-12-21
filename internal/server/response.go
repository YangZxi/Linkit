package server

type ApiResponse[T any] struct {
	Msg  string `json:"msg"`
	Data T      `json:"data"`
	Code int    `json:"code"`
}

func Ok[T any](data T, msg string) ApiResponse[T] {
	return ApiResponse[T]{Msg: msg, Data: data, Code: 200}
}

func Fail[T any](msg string, code int) ApiResponse[T] {
	return ApiResponse[T]{Msg: msg, Data: *new(T), Code: code}
}
