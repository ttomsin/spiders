package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"chatgpt-spider/internal/spider"
)

// Server encapsulates the HTTP API server powered by spider.Engine
type Server struct {
	engine *spider.Engine
	port   int
}

// ChatCompletionMessage represents a role/content message in the OpenAI spec
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type,omitempty"` // e.g. "json_object"
}

// ChatCompletionRequest is the OpenAI-standard request body
type ChatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []ChatCompletionMessage `json:"messages"`
	Stream         bool                    `json:"stream"`
	ConversationID string                  `json:"conversation_id,omitempty"`
	Format         string                  `json:"format,omitempty"`
	ResponseFormat *ResponseFormat         `json:"response_format,omitempty"`
	SessionDelete  bool                    `json:"session_delete,omitempty"`
}

// ChatCompletionChoice is part of the non-streaming response
type ChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

// ChatCompletionResponse is the OpenAI-standard non-streaming response
type ChatCompletionResponse struct {
	ID             string                 `json:"id"`
	Object         string                 `json:"object"`
	Created        int64                  `json:"created"`
	Model          string                 `json:"model"`
	ConversationID string                 `json:"conversation_id"`
	Choices        []ChatCompletionChoice `json:"choices"`
}

// StreamDelta represents a token delta in an SSE stream
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// StreamChoice is a choice in the streaming response
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamChunk is the OpenAI-standard SSE chunk
type StreamChunk struct {
	ID             string         `json:"id"`
	Object         string         `json:"object"`
	Created        int64          `json:"created"`
	Model          string         `json:"model"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Choices        []StreamChoice `json:"choices"`
}

// NewServer creates a new API server
func NewServer(engine *spider.Engine, port int) *Server {
	return &Server{
		engine: engine,
		port:   port,
	}
}

// Start registers routes and starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/history", s.handleHistory)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	models := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{"id": "gpt-4o", "object": "model", "owned_by": "openai"},
			{"id": "chatgpt-spider", "object": "model", "owned_by": "chatgpt-spider"},
		},
	}
	_ = json.NewEncoder(w).Encode(models)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	turns, err := s.engine.GetHistory()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": s.engine.GetCurrentConversationID(),
		"messages":        turns,
	})
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "invalid json: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, `{"error": "messages cannot be empty"}`, http.StatusBadRequest)
		return
	}

	// Extract the prompt from the last user message
	var promptText string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			promptText = req.Messages[i].Content
			break
		}
	}
	if promptText == "" {
		promptText = req.Messages[len(req.Messages)-1].Content
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gpt-4o"
	}

	convID := req.ConversationID
	if convID == "" {
		// Optional header: X-Conversation-ID
		convID = r.Header.Get("X-Conversation-ID")
	}

	formatConstraint := req.Format
	if formatConstraint == "" && req.ResponseFormat != nil {
		if req.ResponseFormat.Type == "json_object" {
			formatConstraint = "json"
		} else if req.ResponseFormat.Type != "" {
			formatConstraint = req.ResponseFormat.Type
		}
	}

	// Handle Streaming SSE response
	if req.Stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())

		// First delta role announcement
		firstChunk := StreamChunk{
			ID:             chunkID,
			Object:         "chat.completion.chunk",
			Created:        time.Now().Unix(),
			Model:          modelName,
			ConversationID: convID,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: StreamDelta{
						Role: "assistant",
					},
				},
			},
		}
		data, _ := json.Marshal(firstChunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		resp, err := s.engine.Prompt(r.Context(), spider.PromptRequest{
			Prompt:         promptText,
			ConversationID: convID,
			Format:         formatConstraint,
			OnToken: func(delta string) {
				currentConvID := convID
				if currentConvID == "" {
					currentConvID = s.engine.GetCurrentConversationID()
				}
				chunk := StreamChunk{
					ID:             chunkID,
					Object:         "chat.completion.chunk",
					Created:        time.Now().Unix(),
					Model:          modelName,
					ConversationID: currentConvID,
					Choices: []StreamChoice{
						{
							Index: 0,
							Delta: StreamDelta{
								Content: delta,
							},
						},
					},
				}
				cData, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", cData)
				flusher.Flush()
			},
		})

		if err != nil {
			errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errPayload)
			flusher.Flush()
			return
		}

		stop := "stop"
		finalChunk := StreamChunk{
			ID:             chunkID,
			Object:         "chat.completion.chunk",
			Created:        time.Now().Unix(),
			Model:          modelName,
			ConversationID: resp.ConversationID,
			Choices: []StreamChoice{
				{
					Index:        0,
					Delta:        StreamDelta{},
					FinishReason: &stop,
				},
			},
		}
		fData, _ := json.Marshal(finalChunk)
		fmt.Fprintf(w, "data: %s\n\n", fData)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()

		if req.SessionDelete && resp.ConversationID != "" {
			_ = s.engine.DeleteConversation(resp.ConversationID)
		}
		return
	}

	// Handle standard non-streaming JSON response
	w.Header().Set("Content-Type", "application/json")
	resp, err := s.engine.Prompt(r.Context(), spider.PromptRequest{
		Prompt:         promptText,
		ConversationID: convID,
		Format:         formatConstraint,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	jsonResp := ChatCompletionResponse{
		ID:             fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:         "chat.completion",
		Created:        time.Now().Unix(),
		Model:          modelName,
		ConversationID: resp.ConversationID,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatCompletionMessage{
					Role:    "assistant",
					Content: resp.Text,
				},
				FinishReason: "stop",
			},
		},
	}

	_ = json.NewEncoder(w).Encode(jsonResp)

	if req.SessionDelete && resp.ConversationID != "" {
		_ = s.engine.DeleteConversation(resp.ConversationID)
	}
}
