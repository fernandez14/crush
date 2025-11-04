package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// Agent defines the interface for handling ACP requests
type Agent interface {
	// Initialize establishes the connection and negotiates capabilities
	Initialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error)
	
	// Authenticate performs authentication (optional)
	Authenticate(ctx context.Context, req AuthenticateRequest) (*AuthenticateResponse, error)
	
	// NewSession creates a new session
	NewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error)
	
	// LoadSession loads an existing session
	LoadSession(ctx context.Context, req LoadSessionRequest) (*LoadSessionResponse, error)
	
	// Prompt handles a user prompt
	Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error)
	
	// Cancel cancels ongoing operations in a session
	Cancel(ctx context.Context, notif CancelNotification) error
}

// Server implements an ACP agent server over stdio
type Server struct {
	agent  Agent
	input  io.Reader
	output io.Writer
	
	mu           sync.Mutex
	nextID       int64
	sendNotifMu  sync.Mutex
}

// NewServer creates a new ACP server
func NewServer(agent Agent, input io.Reader, output io.Writer) *Server {
	return &Server{
		agent:  agent,
		input:  input,
		output: output,
		nextID: 1,
	}
}

// Run starts the server and processes messages from stdin
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.input)
	
	// Increase buffer size for large messages
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		
		// Try to parse as a request or notification
		if err := s.handleMessage(ctx, line); err != nil {
			slog.Error("Failed to handle message", "error", err)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	
	return nil
}

// handleMessage processes a single JSON-RPC message
func (s *Server) handleMessage(ctx context.Context, data []byte) error {
	// First, detect if it has an ID (request) or not (notification)
	var msg struct {
		ID     *RequestID `json:"id"`
		Method string     `json:"method"`
	}
	
	if err := json.Unmarshal(data, &msg); err != nil {
		s.sendError(nil, NewJSONRPCError(ParseError, "parse error", nil))
		return err
	}
	
	if msg.ID != nil {
		// It's a request
		var req JSONRPCRequest
		if err := json.Unmarshal(data, &req); err != nil {
			s.sendError(msg.ID, NewJSONRPCError(ParseError, "parse error", nil))
			return err
		}
		return s.handleRequest(ctx, &req)
	}
	
	// It's a notification
	var notif JSONRPCNotification
	if err := json.Unmarshal(data, &notif); err != nil {
		return err
	}
	return s.handleNotification(ctx, &notif)
}

// handleRequest handles a JSON-RPC request
func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) error {
	var result interface{}
	var err error
	
	switch req.Method {
	case "initialize":
		var params InitializeRequest
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				s.sendError(req.ID, NewInvalidParamsError(err.Error()))
				return nil
			}
		}
		result, err = s.agent.Initialize(ctx, params)
		
	case "authenticate":
		var params AuthenticateRequest
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				s.sendError(req.ID, NewInvalidParamsError(err.Error()))
				return nil
			}
		}
		result, err = s.agent.Authenticate(ctx, params)
		
	case "session/new":
		var params NewSessionRequest
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				s.sendError(req.ID, NewInvalidParamsError(err.Error()))
				return nil
			}
		}
		result, err = s.agent.NewSession(ctx, params)
		
	case "session/load":
		var params LoadSessionRequest
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				s.sendError(req.ID, NewInvalidParamsError(err.Error()))
				return nil
			}
		}
		result, err = s.agent.LoadSession(ctx, params)
		
	case "session/prompt":
		var params PromptRequest
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				s.sendError(req.ID, NewInvalidParamsError(err.Error()))
				return nil
			}
		}
		result, err = s.agent.Prompt(ctx, params)
		
	default:
		s.sendError(req.ID, NewMethodNotFoundError(req.Method))
		return nil
	}
	
	if err != nil {
		s.sendError(req.ID, NewInternalError(err.Error()))
		return nil
	}
	
	return s.sendResult(req.ID, result)
}

// handleNotification handles a JSON-RPC notification
func (s *Server) handleNotification(ctx context.Context, notif *JSONRPCNotification) error {
	switch notif.Method {
	case "session/cancel":
		var params CancelNotification
		if notif.Params != nil {
			if err := json.Unmarshal(notif.Params, &params); err != nil {
				return err
			}
		}
		return s.agent.Cancel(ctx, params)
		
	default:
		slog.Warn("Unknown notification method", "method", notif.Method)
		return nil
	}
}

// sendResult sends a successful JSON-RPC response
func (s *Server) sendResult(id *RequestID, result interface{}) error {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	
	resp := JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultBytes,
	}
	
	return s.sendResponse(&resp)
}

// sendError sends an error JSON-RPC response
func (s *Server) sendError(id *RequestID, jsonrpcErr *JSONRPCError) error {
	resp := JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   jsonrpcErr,
	}
	
	return s.sendResponse(&resp)
}

// sendResponse writes a JSON-RPC response to stdout
func (s *Server) sendResponse(resp *JSONRPCResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	
	data = append(data, '\n')
	_, err = s.output.Write(data)
	return err
}

// SendNotification sends a notification to the client
func (s *Server) SendNotification(method string, params interface{}) error {
	s.sendNotifMu.Lock()
	defer s.sendNotifMu.Unlock()
	
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	
	notif := JSONRPCNotification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsBytes,
	}
	
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	
	data = append(data, '\n')
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, err = s.output.Write(data)
	return err
}

// SendSessionUpdate sends a session/update notification to the client
func (s *Server) SendSessionUpdate(sessionID SessionID, updateType string, updateData interface{}) error {
	dataBytes, err := json.Marshal(updateData)
	if err != nil {
		return err
	}
	
	update := SessionUpdate{
		Type: updateType,
		Data: dataBytes,
	}
	
	notif := SessionNotification{
		SessionID: sessionID,
		Update:    update,
	}
	
	return s.SendNotification("session/update", notif)
}
