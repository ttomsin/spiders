package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"x-spider-ai/pkg/xspiderai"
)

// HTTPServer provides REST / JSON API endpoints for AI systems
type HTTPServer struct {
	client *xspiderai.Client
	port   int
}

func NewHTTPServer(client *xspiderai.Client, port int) *HTTPServer {
	if port <= 0 {
		port = 8080
	}
	return &HTTPServer{
		client: client,
		port:   port,
	}
}

// Start launches the HTTP server
func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "x-spider-ai"})
	})

	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/tweets/post", s.handlePostTweet)
	mux.HandleFunc("/api/v1/tweets/like", s.handleLike)
	mux.HandleFunc("/api/v1/tweets/unlike", s.handleUnlike)
	mux.HandleFunc("/api/v1/tweets/retweet", s.handleRetweet)
	mux.HandleFunc("/api/v1/tweets/bookmark", s.handleBookmark)
	mux.HandleFunc("/api/v1/tweets/delete", s.handleDelete)
	mux.HandleFunc("/api/v1/tweets/thread", s.handleThread)
	mux.HandleFunc("/api/v1/tweets/search", s.handleSearch)
	mux.HandleFunc("/api/v1/users/follow", s.handleFollow)
	mux.HandleFunc("/api/v1/users/unfollow", s.handleUnfollow)
	mux.HandleFunc("/api/v1/users/profile", s.handleProfile)
	mux.HandleFunc("/api/v1/users/me", s.handleMe)
	mux.HandleFunc("/api/v1/users/timeline", s.handleUserTimeline)
	mux.HandleFunc("/api/v1/messages/send", s.handleSendDM)
	mux.HandleFunc("/api/v1/browser/scroll", s.handleScroll)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("[x-spider-ai] HTTP/JSON API server listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		AuthToken string `json:"auth_token"`
		CT0       string `json:"ct0"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	if err := s.client.Login(req.AuthToken, req.CT0); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "authenticated"})
}

func (s *HTTPServer) handlePostTweet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var opts xspiderai.PostTweetOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	res, err := s.client.PostTweet(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleLike(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.LikeTweet(req.TweetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleUnlike(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.UnlikeTweet(req.TweetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleRetweet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.Retweet(req.TweetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleBookmark(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.BookmarkTweet(req.TweetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.DeleteTweet(req.TweetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleThread(w http.ResponseWriter, r *http.Request) {
	tweetID := r.URL.Query().Get("tweet_id")
	if tweetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing tweet_id query param"})
		return
	}
	res, err := s.client.ReadThread(tweetID, xspiderai.ScrollOptions{MaxScrolls: 4})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	tab := r.URL.Query().Get("tab")
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing query param"})
		return
	}
	res, err := s.client.SearchTweets(query, tab, xspiderai.ScrollOptions{MaxScrolls: 3})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleUserTimeline(w http.ResponseWriter, r *http.Request) {
	screenName := r.URL.Query().Get("screen_name")
	if screenName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing screen_name param"})
		return
	}
	res, err := s.client.ReadUserTimeline(screenName, xspiderai.ScrollOptions{MaxScrolls: 3})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleFollow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScreenName string `json:"screen_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.FollowUser(req.ScreenName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScreenName string `json:"screen_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.UnfollowUser(req.ScreenName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleProfile(w http.ResponseWriter, r *http.Request) {
	screenName := r.URL.Query().Get("screen_name")
	if screenName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing screen_name param"})
		return
	}
	prof, err := s.client.GetProfile(screenName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, prof)
}

func (s *HTTPServer) handleMe(w http.ResponseWriter, r *http.Request) {
	prof, err := s.client.GetMyProfile()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, prof)
}

func (s *HTTPServer) handleSendDM(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScreenName string `json:"screen_name"`
		Text       string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := s.client.SendDirectMessage(req.ScreenName, req.Text)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *HTTPServer) handleScroll(w http.ResponseWriter, r *http.Request) {
	res, err := s.client.Scroll(xspiderai.ScrollOptions{MaxScrolls: 2, DelayMs: 1500})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
