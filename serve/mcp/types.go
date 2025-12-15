package mcp

import "encoding/json"

const (
	jsonrpcVersion = "2.0"
)

// Reference (align with MCP Streamable HTTP + JSON-RPC semantics):
// - Requests: {"jsonrpc":"2.0","id":...,"method":"tools/list","params":{...}}
// - Notifications: same but without "id"

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

func newResult(id json.RawMessage, result interface{}) jsonrpcResponse {
	return jsonrpcResponse{JSONRPC: jsonrpcVersion, ID: id, Result: result}
}

func newError(id json.RawMessage, code int, message string, data interface{}) jsonrpcResponse {
	return jsonrpcResponse{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
