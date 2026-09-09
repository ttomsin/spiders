package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"chatgpt-spider/internal/spider"
)

// Server encapsulates the HTTP API server powered by spider.Engine
type Server struct {
	engine               *spider.Engine
	port                 int
	defaultTemporaryChat bool
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
	TemporaryChat  bool                    `json:"temporary_chat,omitempty"`
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
func NewServer(engine *spider.Engine, port int, defaultTemporaryChat bool) *Server {
	return &Server{
		engine:               engine,
		port:                 port,
		defaultTemporaryChat: defaultTemporaryChat,
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
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Conversation-ID")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[SERVER] Error decoding request: %v\n", err)
		http.Error(w, fmt.Sprintf(`{"error": "invalid json: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Because JSON unmarshaling can't distinguish between false and missing for booleans without pointers,
	// we assume the user wants the server default unless it was sent via the API.
	// Since OpenAI spec has no temporary_chat, it will always be missing, so we safely apply the default.
	req.TemporaryChat = s.defaultTemporaryChat

	fmt.Printf("[SERVER] Received %s request for model %s with %d messages (Stream: %v)\n", r.Method, req.Model, len(req.Messages), req.Stream)

	if len(req.Messages) == 0 {
		http.Error(w, `{"error": "messages cannot be empty"}`, http.StatusBadRequest)
		return
	}

	// Since the backend is a web UI chat box, we must compile the stateless API message history into a single string
	var promptBuilder strings.Builder
	
	// If it's a temporary chat, we add our strict stateless directive
	if req.TemporaryChat {
		promptBuilder.WriteString("SYSTEM DIRECTIVE: You are a raw, stateless text-completion API model responding to an IDE. Tool execution (like Python or Advanced Data Analysis) is STRICTLY PROHIBITED. Do not create physical files. Only output raw markdown code blocks. Provide direct outputs with zero conversational filler.\n\n")
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			promptBuilder.WriteString("[System Instruction]: " + msg.Content + "\n\n")
		} else if msg.Role == "assistant" {
			promptBuilder.WriteString("[Assistant]: " + msg.Content + "\n\n")
		} else if msg.Role == "user" {
			promptBuilder.WriteString("[User]: " + msg.Content + "\n\n")
		}
	}
	
	// Tell the model to respond to the last user message
	promptBuilder.WriteString("\nRespond ONLY as the Assistant to the final User message above.")
	
	promptText := promptBuilder.String()

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

		fmt.Printf("[SERVER] Forwarding streaming prompt to engine (TemporaryChat: %v, NewChat: %v)...\n", req.TemporaryChat, convID == "")
		resp, err := s.engine.Prompt(r.Context(), spider.PromptRequest{
			Prompt:         promptText,
			ConversationID: convID,
			Format:         formatConstraint,
			TemporaryChat:  req.TemporaryChat,
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
			fmt.Printf("[SERVER] Engine Prompt Error (Streaming): %v\n", err)
			errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errPayload)
			flusher.Flush()
			return
		}
		fmt.Printf("[SERVER] Streaming completed successfully. ConversationID: %s\n", resp.ConversationID)

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
		TemporaryChat:  req.TemporaryChat,
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
