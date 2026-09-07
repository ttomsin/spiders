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
	"chatgpt-spider/internal/interceptor"
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
	ConversationID string
	NewChat        bool
	OnToken        func(delta string)
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
	inst        *browser.BrowserInstance
	itc         *interceptor.Interceptor
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

	page := inst.Page
	itc, err := interceptor.NewInterceptor(page)
	if err != nil {
		_ = inst.Close()
		return nil, fmt.Errorf("failed to setup interceptor: %w", err)
	}

	eng := &Engine{
		inst: inst,
		itc:  itc,
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
		e.currentConv = initialConvID
		return conversation.OpenConversation(page, initialConvID)
	}

	if err := page.Navigate("https://chatgpt.com"); err != nil {
		return err
	}
	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)
	return nil
}

// Prompt sends a message and returns the response, invoking onToken live as tokens arrive
func (e *Engine) Prompt(ctx context.Context, req PromptRequest) (*PromptResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	page := e.inst.Page

	if req.NewChat {
		if err := conversation.NewChat(page); err == nil {
			e.currentConv = ""
		}
	} else if req.ConversationID != "" && req.ConversationID != e.currentConv {
		if err := conversation.OpenConversation(page, req.ConversationID); err == nil {
			e.currentConv = req.ConversationID
		}
	}

	// Purge stale interceptor chunks before sending
	e.itc.Drain()

	if err := conversation.SendPrompt(page, req.Prompt); err != nil {
		return nil, fmt.Errorf("error sending prompt: %w", err)
	}

	respCh := make(chan string, 1)
	go func() {
		select {
		case resp := <-e.itc.ResponseChannel():
			select {
			case respCh <- resp:
			default:
			}
		case <-time.After(60 * time.Second):
		}
	}()

	// Stream live tokens through the DOM watcher
	go func() {
		if text, err := conversation.StreamCompletion(page, 45*time.Second, req.OnToken); err == nil && text != "" {
			select {
			case respCh <- text:
			default:
			}
		}
	}()

	var responseText string
	var convID string

	select {
	case res := <-respCh:
		responseText = res
		_, convID = e.itc.GetLastResponse()
		if convID == "" {
			convID = conversation.GetCurrentConversationID(page)
		}
		if convID != "" {
			e.currentConv = convID
		}
	case <-time.After(60 * time.Second):
		return nil, fmt.Errorf("response timeout: took longer than 60s")
	}

	return &PromptResponse{
		Text:           responseText,
		ConversationID: e.currentConv,
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
	e.currentConv = convID
	return conversation.OpenConversation(e.inst.Page, convID)
}

// NewChat starts a clean conversation thread
func (e *Engine) NewChat() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.currentConv = ""
	return conversation.NewChat(e.inst.Page)
}

// GetCurrentConversationID returns the current active conversation ID
func (e *Engine) GetCurrentConversationID() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.currentConv != "" {
		return e.currentConv
	}
	return conversation.GetCurrentConversationID(e.inst.Page)
}

// Close gracefully closes the engine
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var itcErr, instErr error
	if e.itc != nil {
		itcErr = e.itc.Stop()
	}
	if e.inst != nil {
		instErr = e.inst.Close()
	}
	if itcErr != nil {
		return itcErr
	}
	return instErr
}
