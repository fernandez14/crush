package acp

import "encoding/json"

// Protocol version constant
const (
	ProtocolVersion = "1.0.0"
)

// Implementation describes client or agent information
type Implementation struct {
	Name    string  `json:"name"`
	Title   *string `json:"title,omitempty"`
	Version string  `json:"version"`
}

// SessionID identifies a session
type SessionID string

// InitializeRequest is sent by the client to establish connection
type InitializeRequest struct {
	ProtocolVersion    string              `json:"protocolVersion"`
	ClientCapabilities *ClientCapabilities `json:"clientCapabilities,omitempty"`
	ClientInfo         *Implementation     `json:"clientInfo,omitempty"`
}

// InitializeResponse is the agent's response to initialization
type InitializeResponse struct {
	ProtocolVersion   string             `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities  `json:"agentCapabilities"`
	AuthMethods       []AuthMethod       `json:"authMethods,omitempty"`
	AgentInfo         *Implementation    `json:"agentInfo,omitempty"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	FS       *FileSystemCapability `json:"fs,omitempty"`
	Terminal bool                  `json:"terminal,omitempty"`
}

// FileSystemCapability describes file system capabilities
type FileSystemCapability struct {
	ReadTextFile  bool `json:"readTextFile"`
	WriteTextFile bool `json:"writeTextFile"`
}

// AgentCapabilities describes what the agent supports
type AgentCapabilities struct {
	Prompt *PromptCapabilities `json:"prompt,omitempty"`
	MCP    *MCPCapabilities    `json:"mcp,omitempty"`
}

// PromptCapabilities describes prompt-related capabilities
type PromptCapabilities struct {
	SupportsResources bool `json:"supportsResources,omitempty"`
	SupportsImages    bool `json:"supportsImages,omitempty"`
}

// MCPCapabilities describes MCP-related capabilities  
type MCPCapabilities struct {
	Servers []MCPServer `json:"servers,omitempty"`
}

// MCPServer describes an MCP server
type MCPServer struct {
	Name string `json:"name"`
}

// AuthMethod describes an authentication method
type AuthMethod struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	Description *string `json:"description,omitempty"`
}

// AuthenticateRequest requests authentication
type AuthenticateRequest struct {
	AuthMethodID string `json:"authMethodId"`
}

// AuthenticateResponse is the response to authentication
type AuthenticateResponse struct{}

// NewSessionRequest creates a new session
type NewSessionRequest struct {
	WorkingDirectory *string `json:"workingDirectory,omitempty"`
}

// NewSessionResponse returns the new session ID
type NewSessionResponse struct {
	SessionID SessionID `json:"sessionId"`
}

// LoadSessionRequest loads an existing session
type LoadSessionRequest struct {
	SessionID SessionID `json:"sessionId"`
}

// LoadSessionResponse confirms session loading
type LoadSessionResponse struct{}

// ContentBlock represents a block of content in a prompt
type ContentBlock struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// TextContent represents text content
type TextContent struct {
	Text string `json:"text"`
}

// PromptRequest sends a user prompt to the agent
type PromptRequest struct {
	SessionID SessionID      `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// PromptResponse is the agent's response to a prompt
type PromptResponse struct {
	StopReason string `json:"stopReason"`
}

// StopReason constants
const (
	StopReasonEndTurn     = "endTurn"
	StopReasonMaxTokens   = "maxTokens"
	StopReasonToolUse     = "toolUse"
	StopReasonError       = "error"
	StopReasonCanceled    = "canceled"
)

// SessionUpdate describes an update to send to the client
type SessionUpdate struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// SessionNotification is sent by the agent to update session state
type SessionNotification struct {
	SessionID SessionID     `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
}

// ContentChunk represents a chunk of content being streamed
type ContentChunk struct {
	Content string `json:"content"`
}

// CancelNotification is sent by the client to cancel operations
type CancelNotification struct {
	SessionID SessionID `json:"sessionId"`
}

// ReadTextFileRequest requests reading a text file
type ReadTextFileRequest struct {
	Path string `json:"path"`
}

// ReadTextFileResponse returns file contents
type ReadTextFileResponse struct {
	Contents string `json:"contents"`
}

// WriteTextFileRequest requests writing a text file
type WriteTextFileRequest struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

// WriteTextFileResponse confirms file write
type WriteTextFileResponse struct{}
