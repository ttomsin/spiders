package crawler

import (
	"net/http"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"x-spider/internal/config"
)

// NetworkInterceptor manages request filtering and Twitter API response extraction
type NetworkInterceptor struct {
	router      *rod.HijackRouter
	cfg         *config.Config
	dataChan    chan []byte
	rateLimChan chan struct{}
}

// NewNetworkInterceptor creates a router to intercept SearchTimeline/TweetDetail and block media
func NewNetworkInterceptor(page *rod.Page, cfg *config.Config) (*NetworkInterceptor, error) {
	router := page.HijackRequests()

	interceptor := &NetworkInterceptor{
		router:      router,
		cfg:         cfg,
		dataChan:    make(chan []byte, 100),
		rateLimChan: make(chan struct{}, 10),
	}

	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqURL := ctx.Request.URL().String()

		// 1. Block heavy images and videos
		for _, ext := range cfg.BlockedExtensions {
			if strings.Contains(strings.ToLower(reqURL), strings.ToLower(ext)) {
				ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
				return
			}
		}

		// 2. Intercept Twitter GraphQL timeline and thread responses
		if strings.Contains(reqURL, "SearchTimeline") || strings.Contains(reqURL, "TweetDetail") {
			// Load full response from the server
			if err := ctx.LoadResponse(http.DefaultClient, true); err == nil {
				body := ctx.Response.Body()
				payload := ctx.Response.Payload()

				// Detect genuine Twitter HTTP 429 Too Many Requests
				if payload != nil && payload.ResponseCode == 429 {
					select {
					case interceptor.rateLimChan <- struct{}{}:
					default:
					}
					return
				}

				// Check for explicit GraphQL rate limit errors (code 88 or "Rate limit exceeded" inside "errors")
				// Only trigger if it's an actual API error, not a tweet mentioning the words "rate limit"
				if strings.Contains(body, `"errors"`) && (strings.Contains(body, `"code":88`) || strings.Contains(body, "Rate limit exceeded")) {
					select {
					case interceptor.rateLimChan <- struct{}{}:
					default:
					}
					return
				}

				select {
				case interceptor.dataChan <- []byte(body):
				default:
				}
			}
			return
		}

		// 3. Continue all other requests normally
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	return interceptor, nil
}

// DataChannel returns the channel receiving raw JSON responses from Twitter
func (n *NetworkInterceptor) DataChannel() <-chan []byte {
	return n.dataChan
}

// RateLimitChannel notifies when a rate limit message is intercepted
func (n *NetworkInterceptor) RateLimitChannel() <-chan struct{} {
	return n.rateLimChan
}

// Stop terminates the network router
func (n *NetworkInterceptor) Stop() error {
	return n.router.Stop()
}
