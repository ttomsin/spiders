package spider

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"chatgpt-spider/internal/browser"
	"chatgpt-spider/internal/conversation"
	"chatgpt-spider/internal/format"
)

// Options holds initialization options for the spider engine
type Options struct {
	Headless     bool
	Anonymous    bool
	SessionToken string
	Debug        bool
}

// PromptRequest defines a request to ChatGPT
type PromptRequest struct {
	Prompt         string
	FileContent    string // Optional large text/file attachment
	ConversationID string
	NewChat        bool
	Format         string // e.g. "json", "csv", "xml", or custom schema
	OnToken        func(delta string)
	Timeout        time.Duration // Optional custom generation timeout (defaults to 5 minutes)
}

// PromptResponse defines the response from ChatGPT
type PromptResponse struct {
	Text           string
	ConversationID string
	Timestamp      string
}

// Engine encapsulates the browser and automation lifecycle
type Engine struct {
	mu          sync.Mutex
	convMu      sync.RWMutex
	inst        *browser.BrowserInstance
	currentConv string
}

// NewEngine creates and connects a new spider engine
func NewEngine(opts Options) (*Engine, error) {
	inst, err := browser.Launch(browser.Options{
		Headless:     opts.Headless,
		Anonymous:    opts.Anonymous,
		SessionToken: opts.SessionToken,
		Debug:        opts.Debug,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	eng := &Engine{
		inst: inst,
	}

	// Graceful termination handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		_ = eng.Close()
		os.Exit(0)
	}()

	return eng, nil
}

// Initialize opens ChatGPT home or a specific conversation
func (e *Engine) Initialize(initialConvID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	page := e.inst.Page
	if initialConvID != "" {
		e.setConversationID(initialConvID)
		return conversation.OpenConversation(page, initialConvID)
	}

	if err := page.Navigate("https://chatgpt.com"); err != nil {
		return err
	}
	_ = page.WaitLoad()
	return conversation.WaitUntilReady(page, 6*time.Second)
}

// Prompt sends a message and returns the response, invoking onToken live as tokens arrive
func (e *Engine) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	page := e.inst.Page

	if req.NewChat {
		if err := conversation.NewChat(page); err == nil {
			e.setConversationID("")
		}
	} else if req.ConversationID != "" && req.ConversationID != e.GetCurrentConversationID() {
		if err := conversation.OpenConversation(page, req.ConversationID); err == nil {
			e.setConversationID(req.ConversationID)
		}
	}

	// Count assistant messages in DOM before dispatching prompt
	initialCount := conversation.CountAssistantMessages(page)

	effectivePrompt := req.Prompt
	if req.Format != "" {
		effectivePrompt = format.ApplyFormatConstraint(req.Prompt, req.Format)
	}

	if err := conversation.SendPrompt(ctx, page, effectivePrompt, req.FileContent); err != nil {
		return nil, fmt.Errorf("error sending prompt: %w", err)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	// Stream live tokens through the DOM watcher
	responseText, err := conversation.StreamCompletion(ctx, page, initialCount, timeout, req.OnToken)
	if err != nil {
		return nil, err
	}

	// Read updated conversation ID from page URL
	convID := conversation.GetCurrentConversationID(page)
	if convID != "" {
		e.setConversationID(convID)
	}

	return &PromptResponse{
		Text:           responseText,
		ConversationID: e.GetCurrentConversationID(),
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// GetHistory retrieves the conversation messages for the active conversation
func (e *Engine) GetHistory() ([]conversation.ConversationTurn, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return conversation.GetConversationHistory(e.inst.Page)
}

// SwitchConversation changes the active thread
func (e *Engine) SwitchConversation(convID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setConversationID(convID)
	return conversation.OpenConversation(e.inst.Page, convID)
}

// NewChat starts a clean conversation thread
func (e *Engine) NewChat() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setConversationID("")
	return conversation.NewChat(e.inst.Page)
}

// DeleteConversation deletes a conversation by ID from the account and resets the session
func (e *Engine) DeleteConversation(convID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	targetID := convID
	if targetID == "" {
		targetID = e.GetCurrentConversationID()
	}
	if targetID == "" {
		return nil
	}
	err := conversation.DeleteConversation(e.inst.Page, targetID)
	if e.GetCurrentConversationID() == targetID {
		e.setConversationID("")
	}
	return err
}

// GetCurrentConversationID returns the current active conversation ID (non-blocking)
func (e *Engine) GetCurrentConversationID() string {
	e.convMu.RLock()
	id := e.currentConv
	e.convMu.RUnlock()
	if id != "" {
		return id
	}
	if e.inst != nil && e.inst.Page != nil {
		return conversation.GetCurrentConversationID(e.inst.Page)
	}
	return ""
}

func (e *Engine) setConversationID(id string) {
	e.convMu.Lock()
	e.currentConv = id
	e.convMu.Unlock()
}

// Close gracefully closes the engine
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.inst != nil {
		return e.inst.Close()
	}
	return nil
}
