package conversation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// sseChunk represents the various JSON formats ChatGPT uses in its SSE stream.
// ChatGPT uses a JSON-Patch-like protocol with three distinct formats:
//
//	Format 1 (append):  {"p":"/message/content/parts/0","o":"append","v":"text chunk"}
//	Format 2 (delta):   {"v":"text chunk"}   (bare string delta, no p/o)
//	Format 3 (patch):   {"p":"","o":"patch","v":[{"p":"/message/content/parts/0","o":"append","v":"text"},...]}
type sseChunk struct {
	P string          `json:"p"`
	O string          `json:"o"`
	V json.RawMessage `json:"v"`
}

// HijackInterceptor intercepts ChatGPT's SSE stream at the Chrome DevTools Protocol (CDP)
// Fetch domain level. Unlike the window.fetch JS hook approach, CDP interception operates
// BELOW JavaScript — React re-renders, service worker updates, and frontend changes cannot
// overwrite or break it.
type HijackInterceptor struct {
	mu        sync.Mutex
	tokenChan chan string
	doneChan  chan struct{}
	router    *rod.HijackRouter
}

// NewHijackInterceptor creates a HijackInterceptor and immediately starts the router on the given page.
// The router runs for the lifetime of the page and persists across individual prompt requests.
func NewHijackInterceptor(page *rod.Page) *HijackInterceptor {
	h := &HijackInterceptor{
		tokenChan: make(chan string, 10000),
		doneChan:  make(chan struct{}, 1),
	}

	h.router = page.HijackRequests()

	// Match both the classic and the newer /f/ conversation endpoints
	for _, pattern := range []string{
		"*chatgpt.com/backend-api/f/conversation",
		"*chatgpt.com/backend-api/conversation",
	} {
		pat := pattern
		h.router.MustAdd(pat, func(ctx *rod.Hijack) {
			// Only intercept POST (the actual prompt request)
			if ctx.Request.Method() != "POST" {
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}
			// Skip ancillary sub-resources
			u := ctx.Request.URL().String()
			if strings.Contains(u, "stream_status") ||
				strings.Contains(u, "textdocs") ||
				strings.Contains(u, "/prepare") {
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}

			// Snapshot current channels (set by the most recent Reset() call)
			h.mu.Lock()
			tokenCh := h.tokenChan
			doneCh := h.doneChan
			h.mu.Unlock()

			// Build a direct HTTP request using the browser's auth headers & cookies
			reqBody := ctx.Request.Body()
			httpReq, err := http.NewRequest("POST", u, bytes.NewReader([]byte(reqBody)))
			if err != nil {
				fmt.Printf("[HIJACK] Failed to create request: %v\n", err)
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}
			for k, v := range ctx.Request.Headers() {
				httpReq.Header.Set(k, v.String())
			}

			// Execute the streaming request (this blocks until ChatGPT finishes generating)
			client := &http.Client{Timeout: 10 * time.Minute}
			resp, err := client.Do(httpReq)
			if err != nil {
				fmt.Printf("[HIJACK] Direct stream request failed: %v\n", err)
				ctx.ContinueRequest(&proto.FetchContinueRequest{})
				return
			}
			defer resp.Body.Close()

			// Parse SSE stream line-by-line
			var rawSSEBody strings.Builder
			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

			for scanner.Scan() {
				line := scanner.Text()
				rawSSEBody.WriteString(line + "\n")

				if !strings.HasPrefix(line, "data: ") {
					continue
				}
				dataStr := strings.TrimPrefix(line, "data: ")
				if dataStr == "[DONE]" || dataStr == `"v1"` || dataStr == "v1" {
					continue
				}

				token := ExtractTokenFromSSE(dataStr)
				if token != "" {
					select {
					case tokenCh <- token:
					default:
						// Channel full — drop to avoid blocking the stream reader
					}
				}
			}

			// Signal stream completion
			select {
			case doneCh <- struct{}{}:
			default:
			}

			// Fulfill the browser's paused Fetch request with the complete SSE body.
			// The ChatGPT UI will receive the full response (all at once rather than streamed),
			// but that is an acceptable trade-off since Continue IDE receives real-time tokens.
			ctx.Response.SetBody(rawSSEBody.String())
			ctx.Response.SetHeader("Content-Type", "text/event-stream")
			ctx.Response.SetHeader("Cache-Control", "no-cache")
		})
	}

	go h.router.Run()
	return h
}

// Reset clears the token and done channels in preparation for a new prompt request.
// Call this BEFORE SendPrompt so the channels are fresh when the interceptor fires.
func (h *HijackInterceptor) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tokenChan = make(chan string, 10000)
	h.doneChan = make(chan struct{}, 1)
}

// Channels returns snapshots of the current token and done channels.
// Snapshot them once before StreamCompletionFromHijack so you always read the right generation.
func (h *HijackInterceptor) Channels() (tokenChan chan string, doneChan chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tokenChan, h.doneChan
}

// Stop shuts down the hijack router. Call when the engine is closing.
func (h *HijackInterceptor) Stop() {
	if h.router != nil {
		h.router.Stop()
	}
}

// ExtractTokenFromSSE parses a single SSE data-line and returns the text token contained within it.
// It handles all three JSON formats ChatGPT uses in its streaming protocol.
func ExtractTokenFromSSE(data string) string {
	var chunk sseChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return ""
	}

	// Format 1 — JSON Patch append: {"p":"/message/content/parts/0","o":"append","v":"text"}
	if chunk.O == "append" && chunk.P == "/message/content/parts/0" {
		var s string
		if json.Unmarshal(chunk.V, &s) == nil {
			return s
		}
	}

	// Format 2 — bare delta: {"v":"text"}  (no p or o fields present)
	if chunk.P == "" && chunk.O == "" && len(chunk.V) > 0 {
		var s string
		if json.Unmarshal(chunk.V, &s) == nil {
			return s
		}
	}

	// Format 3 — patch array: {"p":"","o":"patch","v":[{"p":"/message/content/parts/0","o":"append","v":"text"},...]}
	if chunk.O == "patch" {
		var patches []sseChunk
		if json.Unmarshal(chunk.V, &patches) == nil {
			var sb strings.Builder
			for _, p := range patches {
				if p.O == "append" && p.P == "/message/content/parts/0" {
					var s string
					if json.Unmarshal(p.V, &s) == nil {
						sb.WriteString(s)
					}
				}
			}
			return sb.String()
		}
	}

	return ""
}

// StreamCompletionFromHijack reads tokens from the HijackInterceptor channels in real-time
// with zero polling overhead. It blocks until the SSE stream signals completion or the context
// / timeout fires.
func StreamCompletionFromHijack(ctx context.Context, tokenChan chan string, doneChan chan struct{}, maxWait time.Duration, onDelta func(string)) (string, error) {
	if maxWait <= 0 {
		maxWait = 5 * time.Minute
	}
	timeout := time.NewTimer(maxWait)
	defer timeout.Stop()

	// Give the browser up to 20s to start the request (page reload + React init)
	startTimeout := time.NewTimer(20 * time.Second)
	defer startTimeout.Stop()

	var sb strings.Builder
	started := false

	for {
		select {
		case <-ctx.Done():
			return strings.TrimSpace(sb.String()), ctx.Err()

		case token := <-tokenChan:
			if !started {
				started = true
				startTimeout.Stop()
			}
			sb.WriteString(token)
			if onDelta != nil {
				onDelta(token)
			}

		case <-doneChan:
			// Drain any remaining tokens already queued before the done signal
			for {
				select {
				case token := <-tokenChan:
					sb.WriteString(token)
					if onDelta != nil {
						onDelta(token)
					}
				default:
					return strings.TrimSpace(sb.String()), nil
				}
			}

		case <-startTimeout.C:
			if !started {
				return "", fmt.Errorf("hijack: timeout waiting for ChatGPT to begin generating")
			}

		case <-timeout.C:
			if sb.Len() > 0 {
				return strings.TrimSpace(sb.String()), nil
			}
			return "", fmt.Errorf("hijack: timeout waiting for response")
		}
	}
}
