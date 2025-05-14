package lsp

import "llm-language-server/jsonrpc"

type Message struct {
	JsonRPC string `json:"jsonrpc"`
}

type RequestMessage struct {
	Message `json:",inline"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// TODO: error code enum
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type ResponseMessage struct {
	Message `json:",inline"`
	ID      int `json:"id"`
	Result  any `json:"result"`
	Error   ResponseError
}

type Notification struct {
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

type NotificationMessage struct {
	Notification `json:",inline"`
	Params       any `json:"params"`
}

func NewResponseMessage(message ResponseMessage) []byte {
	return []byte(jsonrpc.EncodeMessage(message))
}

func NewNotificationMessage(message NotificationMessage) []byte {
	return []byte(jsonrpc.EncodeMessage(message))
}
