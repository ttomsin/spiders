package interceptor

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Interceptor manages network interception of ChatGPT backend API responses
type Interceptor struct {
	router      *rod.HijackRouter
	mu          sync.Mutex
	lastText    string
	lastConvID  string
	responseCh  chan string
	newPromptCh chan struct{}
}

// NewInterceptor creates a request hijacker for ChatGPT conversations
func NewInterceptor(page *rod.Page) (*Interceptor, error) {
	router := page.HijackRequests()

	itc := &Interceptor{
		router:      router,
		responseCh:  make(chan string, 10),
		newPromptCh: make(chan struct{}, 10),
	}

	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqURL := ctx.Request.URL().String()

		// Intercept POST /backend-api/conversation (or /backend-anon/conversation)
		if (strings.Contains(reqURL, "/backend-api/conversation") || strings.Contains(reqURL, "/backend-anon/conversation")) && ctx.Request.Method() == "POST" {
			// Signal new prompt started
			select {
			case itc.newPromptCh <- struct{}{}:
			default:
			}

			if err := ctx.LoadResponse(http.DefaultClient, true); err == nil {
				body := ctx.Response.Body()
				if text, convID, err := ParseSSEStream([]byte(body)); err == nil && text != "" {
					itc.mu.Lock()
					itc.lastText = text
					itc.lastConvID = convID
					itc.mu.Unlock()

					select {
					case itc.responseCh <- text:
					default:
					}
				}
			}
			return
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()
	return itc, nil
}

// ResponseChannel returns the channel that emits intercepted text responses
func (itc *Interceptor) ResponseChannel() <-chan string {
	return itc.responseCh
}

// Drain flushes any lingering responses from the channel before starting a new turn
func (itc *Interceptor) Drain() {
	itc.mu.Lock()
	defer itc.mu.Unlock()
	for {
		select {
		case <-itc.responseCh:
		default:
			return
		}
	}
}

// GetLastResponse returns the most recent captured assistant response
func (itc *Interceptor) GetLastResponse() (string, string) {
	itc.mu.Lock()
	defer itc.mu.Unlock()
	return itc.lastText, itc.lastConvID
}

// Stop terminates the hijacker
func (itc *Interceptor) Stop() error {
	return itc.router.Stop()
}