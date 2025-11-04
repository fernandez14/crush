package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/charmbracelet/crush/internal/app"
)

// CrushAgent implements the ACP Agent interface using Crush's existing agent infrastructure
type CrushAgent struct {
	app     *app.App
	server  *Server
	
	mu       sync.Mutex
	sessions map[SessionID]*sessionState
}

type sessionState struct {
	crushSessionID string
	workingDir     string
}

// NewCrushAgent creates a new Crush ACP agent
func NewCrushAgent(app *app.App) *CrushAgent {
	return &CrushAgent{
		app:      app,
		sessions: make(map[SessionID]*sessionState),
	}
}

// SetServer sets the ACP server for sending notifications
func (a *CrushAgent) SetServer(server *Server) {
	a.server = server
}

// Initialize implements Agent.Initialize
func (a *CrushAgent) Initialize(ctx context.Context, req InitializeRequest) (*InitializeResponse, error) {
	slog.Info("ACP Initialize", "protocol_version", req.ProtocolVersion, "client_info", req.ClientInfo)
	
	resp := &InitializeResponse{
		ProtocolVersion: ProtocolVersion,
		AgentCapabilities: AgentCapabilities{
			Prompt: &PromptCapabilities{
				SupportsResources: false,
				SupportsImages:    false,
			},
		},
		AuthMethods: []AuthMethod{},
		AgentInfo: &Implementation{
			Name:    "crush",
			Title:   strPtr("Crush AI Coding Agent"),
			Version: "0.1.0", // TODO: Get actual version
		},
	}
	
	return resp, nil
}

// Authenticate implements Agent.Authenticate
func (a *CrushAgent) Authenticate(ctx context.Context, req AuthenticateRequest) (*AuthenticateResponse, error) {
	slog.Info("ACP Authenticate", "auth_method_id", req.AuthMethodID)
	return &AuthenticateResponse{}, nil
}

// NewSession implements Agent.NewSession
func (a *CrushAgent) NewSession(ctx context.Context, req NewSessionRequest) (*NewSessionResponse, error) {
	workingDir := ""
	if req.WorkingDirectory != nil {
		workingDir = *req.WorkingDirectory
	}
	
	slog.Info("ACP NewSession", "working_directory", workingDir)
	
	// Create a new Crush session
	title := "ACP Session"
	if workingDir != "" {
		title = fmt.Sprintf("ACP Session: %s", workingDir)
	}
	
	crushSession, err := a.app.Sessions.Create(ctx, title)
	if err != nil {
		return nil, fmt.Errorf("failed to create Crush session: %w", err)
	}
	
	// Generate ACP session ID
	acpSessionID := SessionID(crushSession.ID)
	
	// Store session state
	a.mu.Lock()
	a.sessions[acpSessionID] = &sessionState{
		crushSessionID: crushSession.ID,
		workingDir:     workingDir,
	}
	a.mu.Unlock()
	
	return &NewSessionResponse{
		SessionID: acpSessionID,
	}, nil
}

// LoadSession implements Agent.LoadSession
func (a *CrushAgent) LoadSession(ctx context.Context, req LoadSessionRequest) (*LoadSessionResponse, error) {
	slog.Info("ACP LoadSession", "session_id", req.SessionID)
	
	// Check if session exists
	a.mu.Lock()
	state, exists := a.sessions[req.SessionID]
	a.mu.Unlock()
	
	if !exists {
		// Try to load from Crush's session service
		crushSessionID := string(req.SessionID)
		_, err := a.app.Sessions.Get(ctx, crushSessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %w", err)
		}
		
		// Add to our tracking
		a.mu.Lock()
		a.sessions[req.SessionID] = &sessionState{
			crushSessionID: crushSessionID,
		}
		a.mu.Unlock()
	} else {
		slog.Debug("Session already loaded", "session_id", req.SessionID, "crush_session_id", state.crushSessionID)
	}
	
	return &LoadSessionResponse{}, nil
}

// Prompt implements Agent.Prompt
func (a *CrushAgent) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	slog.Info("ACP Prompt", "session_id", req.SessionID, "content_blocks", len(req.Prompt))
	
	// Get session state
	a.mu.Lock()
	state, exists := a.sessions[req.SessionID]
	a.mu.Unlock()
	
	if !exists {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	
	// Convert ACP content blocks to Crush prompt
	var promptParts []string
	for _, block := range req.Prompt {
		if block.Type == "text" {
			var textContent TextContent
			if err := json.Unmarshal(block.Data, &textContent); err != nil {
				slog.Warn("Failed to unmarshal text content", "error", err)
				continue
			}
			promptParts = append(promptParts, textContent.Text)
		}
	}
	
	prompt := strings.Join(promptParts, "\n")
	
	// Send start streaming notification
	if err := a.sendAgentMessageStart(req.SessionID); err != nil {
		slog.Error("Failed to send message start notification", "error", err)
	}
	
	// Run the Crush agent coordinator
	result, err := a.app.AgentCoordinator.Run(ctx, state.crushSessionID, prompt)
	if err != nil {
		// Send error notification
		a.sendAgentMessageChunk(req.SessionID, fmt.Sprintf("Error: %v", err))
		return &PromptResponse{
			StopReason: StopReasonError,
		}, nil
	}
	
	// Stream the result back as message chunks
	responseText := result.Response.Content.Text()
	if responseText != "" {
		if err := a.sendAgentMessageChunk(req.SessionID, responseText); err != nil {
			slog.Error("Failed to send message chunk", "error", err)
		}
	}
	
	// Send end streaming notification
	if err := a.sendAgentMessageEnd(req.SessionID); err != nil {
		slog.Error("Failed to send message end notification", "error", err)
	}
	
	return &PromptResponse{
		StopReason: StopReasonEndTurn,
	}, nil
}

// Cancel implements Agent.Cancel
func (a *CrushAgent) Cancel(ctx context.Context, notif CancelNotification) error {
	slog.Info("ACP Cancel", "session_id", notif.SessionID)
	
	// Get session state
	a.mu.Lock()
	state, exists := a.sessions[notif.SessionID]
	a.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("session not found: %s", notif.SessionID)
	}
	
	// Cancel the Crush agent session
	a.app.AgentCoordinator.Cancel(state.crushSessionID)
	
	return nil
}

// Helper methods for sending notifications

func (a *CrushAgent) sendAgentMessageStart(sessionID SessionID) error {
	if a.server == nil {
		return nil
	}
	return a.server.SendSessionUpdate(sessionID, "agentMessageStart", nil)
}

func (a *CrushAgent) sendAgentMessageChunk(sessionID SessionID, content string) error {
	if a.server == nil {
		return nil
	}
	chunk := ContentChunk{
		Content: content,
	}
	return a.server.SendSessionUpdate(sessionID, "agentMessageChunk", chunk)
}

func (a *CrushAgent) sendAgentMessageEnd(sessionID SessionID) error {
	if a.server == nil {
		return nil
	}
	return a.server.SendSessionUpdate(sessionID, "agentMessageEnd", nil)
}

// Helper functions

func strPtr(s string) *string {
	return &s
}
