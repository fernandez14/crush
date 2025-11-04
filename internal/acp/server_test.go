package acp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockAgent is a simple agent for testing
type mockAgent struct {
	initializeResp *InitializeResponse
	sessionCount   int
}

func (m *mockAgent) Initialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
	if m.initializeResp != nil {
		return m.initializeResp, nil
	}
	return &InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		AgentCapabilities: AgentCapabilities{
			Prompt: &PromptCapabilities{},
		},
		AgentInfo: &Implementation{
			Name:    "test-agent",
			Version: "1.0.0",
		},
	}, nil
}

func (m *mockAgent) Authenticate(ctx context.Context, req AuthenticateRequest) (*AuthenticateResponse, error) {
	return &AuthenticateResponse{}, nil
}

func (m *mockAgent) NewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error) {
	m.sessionCount++
	sessionID := SessionID("session-1")
	return &NewSessionResponse{
		SessionID: sessionID,
	}, nil
}

func (m *mockAgent) LoadSession(ctx context.Context, req LoadSessionRequest) (*LoadSessionResponse, error) {
	return &LoadSessionResponse{}, nil
}

func (m *mockAgent) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	return &PromptResponse{
		StopReason: StopReasonEndTurn,
	}, nil
}

func (m *mockAgent) Cancel(ctx context.Context, notif CancelNotification) error {
	return nil
}

// syncBuffer is a thread-safe buffer for testing
type syncBuffer struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func newSyncBuffer() *syncBuffer {
	return &syncBuffer{buf: &bytes.Buffer{}}
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Bytes()
}

func TestServerInitialize(t *testing.T) {
	agent := &mockAgent{}
	input := strings.NewReader(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"1.0.0"}}` + "\n")
	output := newSyncBuffer()
	
	server := NewServer(agent, input, output)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()
	
	// Wait for response with timeout
	time.Sleep(100 * time.Millisecond)
	
	// Get output
	respBytes := output.Bytes()
	if len(respBytes) == 0 {
		t.Fatal("No response received")
	}
	
	// Parse line by line
	lines := bytes.Split(respBytes, []byte("\n"))
	if len(lines) < 1 {
		t.Fatal("No complete response line")
	}
	
	var resp JSONRPCResponse
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v\nResponse: %s", err, string(lines[0]))
	}
	
	if resp.Error != nil {
		t.Fatalf("Got error response: %+v", resp.Error)
	}
	
	var initResp InitializeResponse
	if err := json.Unmarshal(resp.Result, &initResp); err != nil {
		t.Fatalf("Failed to parse initialize response: %v", err)
	}
	
	if initResp.ProtocolVersion != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %s", ProtocolVersion, initResp.ProtocolVersion)
	}
	
	if initResp.AgentInfo == nil || initResp.AgentInfo.Name != "test-agent" {
		t.Error("AgentInfo not set correctly")
	}
}

func TestServerNewSession(t *testing.T) {
	agent := &mockAgent{}
	input := strings.NewReader(`{"jsonrpc":"2.0","id":"1","method":"session/new","params":{}}` + "\n")
	output := newSyncBuffer()
	
	server := NewServer(agent, input, output)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()
	
	// Wait for response
	time.Sleep(100 * time.Millisecond)
	
	respBytes := output.Bytes()
	if len(respBytes) == 0 {
		t.Fatal("No response received")
	}
	
	lines := bytes.Split(respBytes, []byte("\n"))
	if len(lines) < 1 {
		t.Fatal("No complete response line")
	}
	
	var resp JSONRPCResponse
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if resp.Error != nil {
		t.Fatalf("Got error response: %+v", resp.Error)
	}
	
	var sessionResp NewSessionResponse
	if err := json.Unmarshal(resp.Result, &sessionResp); err != nil {
		t.Fatalf("Failed to parse session response: %v", err)
	}
	
	if sessionResp.SessionID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got '%s'", sessionResp.SessionID)
	}
	
	if agent.sessionCount != 1 {
		t.Errorf("Expected 1 session created, got %d", agent.sessionCount)
	}
}

func TestServerPrompt(t *testing.T) {
	agent := &mockAgent{}
	
	promptReq := PromptRequest{
		SessionID: "test-session",
		Prompt: []ContentBlock{
			{
				Type: "text",
				Data: json.RawMessage(`{"text":"Hello world"}`),
			},
		},
	}
	reqBytes, _ := json.Marshal(promptReq)
	
	strID := "1"
	jsonRPCReq := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      &RequestID{Str: &strID},
		Method:  "session/prompt",
		Params:  reqBytes,
	}
	reqLine, _ := json.Marshal(jsonRPCReq)
	
	input := strings.NewReader(string(reqLine) + "\n")
	output := newSyncBuffer()
	
	server := NewServer(agent, input, output)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx)
	}()
	
	// Wait for response
	time.Sleep(100 * time.Millisecond)
	
	respBytes := output.Bytes()
	if len(respBytes) == 0 {
		t.Fatal("No response received")
	}
	
	lines := bytes.Split(respBytes, []byte("\n"))
	if len(lines) < 1 {
		t.Fatal("No complete response line")
	}
	
	var resp JSONRPCResponse
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if resp.Error != nil {
		t.Fatalf("Got error response: %+v", resp.Error)
	}
	
	var promptResp PromptResponse
	if err := json.Unmarshal(resp.Result, &promptResp); err != nil {
		t.Fatalf("Failed to parse prompt response: %v", err)
	}
	
	if promptResp.StopReason != StopReasonEndTurn {
		t.Errorf("Expected stop reason '%s', got '%s'", StopReasonEndTurn, promptResp.StopReason)
	}
}
