// Package acp implements the Agent Client Protocol (ACP) for Crush.
//
// ACP standardizes communication between code editors and AI coding agents,
// enabling Crush to be used as an agent in ACP-compatible IDEs like Zed.
package acp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 constants
const (
	JSONRPCVersion = "2.0"
)

// RequestID represents a JSON-RPC request ID (string or number)
type RequestID struct {
	Str *string
	Num *int64
}

func (r RequestID) MarshalJSON() ([]byte, error) {
	if r.Str != nil {
		return json.Marshal(*r.Str)
	}
	if r.Num != nil {
		return json.Marshal(*r.Num)
	}
	return json.Marshal(nil)
}

func (r *RequestID) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.Str = &s
		return nil
	}
	
	// Try number
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		r.Num = &n
		return nil
	}
	
	return fmt.Errorf("invalid request ID")
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *RequestID      `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *RequestID      `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID)
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// NewJSONRPCError creates a new JSON-RPC error
func NewJSONRPCError(code int, message string, data interface{}) *JSONRPCError {
	err := &JSONRPCError{
		Code:    code,
		Message: message,
	}
	if data != nil {
		dataBytes, _ := json.Marshal(data)
		err.Data = dataBytes
	}
	return err
}

// NewInternalError creates a new internal error
func NewInternalError(message string) *JSONRPCError {
	return NewJSONRPCError(InternalError, message, nil)
}

// NewMethodNotFoundError creates a new method not found error
func NewMethodNotFoundError(method string) *JSONRPCError {
	return NewJSONRPCError(MethodNotFound, fmt.Sprintf("method not found: %s", method), nil)
}

// NewInvalidParamsError creates a new invalid params error
func NewInvalidParamsError(message string) *JSONRPCError {
	return NewJSONRPCError(InvalidParams, message, nil)
}
