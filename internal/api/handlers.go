package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HandleChat handles chat requests
func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check content type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Max allowable payload size (10MB)
	const maxPayloadSize = 10 * 1024 * 1024 // 10 MB

	// Fast-path: if Content-Length header is provided and exceeds the limit return 413 immediately.
	if r.ContentLength > maxPayloadSize {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Wrap body reader to enforce the same limit during streaming reads.
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	// Parse request
	var req ChatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Check if it's a size limit error
		// MaxBytesReader returns a *MaxBytesError when the limit is exceeded
		if strings.Contains(err.Error(), "request body too large") ||
			strings.Contains(err.Error(), "Request entity too large") {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		http.Error(w, "Messages cannot be empty", http.StatusBadRequest)
		return
	}

	// Validate message roles
	for _, msg := range req.Messages {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			http.Error(w, "Invalid message role", http.StatusBadRequest)
			return
		}
	}

	// Create response
	resp := ChatResponse{
		ID: fmt.Sprintf("chat-%d", time.Now().UnixNano()),
		Message: Message{
			Role:    "assistant",
			Content: "This is a response from the API",
		},
		Created: time.Now(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleStream handles streaming chat requests
func (h *Handler) HandleStream(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check content type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Check request size
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	// Parse request
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		http.Error(w, "Messages cannot be empty", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send chunks
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream response chunks
	chunks := []string{"Hello", " from", " the", " API"}
	for _, chunk := range chunks {
		data := StreamChunk{
			ID:      req.ID,
			Content: chunk,
			Done:    false,
		}

		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		time.Sleep(10 * time.Millisecond) // Simulate processing
	}

	// Send completion
	complete := StreamChunk{
		ID:   req.ID,
		Done: true,
	}
	jsonData, _ := json.Marshal(complete)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// HandleTools handles tool execution requests
func (h *Handler) HandleTools(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check content type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Check request size
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	// Parse request
	var req ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate tool name
	if req.Name == "" {
		http.Error(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	// Handle different tools
	switch req.Name {
	case "file_read":
		// Check for path argument
		path, ok := req.Arguments["path"].(string)
		if !ok || path == "" {
			http.Error(w, "Path argument is required for file_read", http.StatusBadRequest)
			return
		}

		// Security check
		if path == "/etc/passwd" {
			w.WriteHeader(http.StatusForbidden)
			resp := ToolResponse{
				ID:    fmt.Sprintf("tool-%d", time.Now().UnixNano()),
				Error: "Access denied: cannot read sensitive files",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Success response
		resp := ToolResponse{
			ID:     fmt.Sprintf("tool-%d", time.Now().UnixNano()),
			Result: "File content would be here",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "Unknown tool: "+req.Name, http.StatusBadRequest)
	}
}

// HandleHistory handles conversation history requests
func (h *Handler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	// Only GET allowed
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Check for specific conversation ID
	if id := query.Get("id"); id != "" {
		if id == "non-existent" {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}

		// Return single conversation
		resp := HistoryResponse{
			Conversations: []Conversation{
				{
					ID:      id,
					Title:   "Test Conversation",
					Created: time.Now().Add(-24 * time.Hour),
					Updated: time.Now(),
					Messages: []Message{
						{Role: "user", Content: "Hello"},
						{Role: "assistant", Content: "Hi there!"},
					},
				},
			},
			Total:   1,
			Page:    1,
			PerPage: 1,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Parse pagination
	page := 1
	perPage := 20

	if p := query.Get("page"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil {
			http.Error(w, "Invalid page number", http.StatusBadRequest)
			return
		}
		page = parsed
	}

	if pp := query.Get("per_page"); pp != "" {
		parsed, err := strconv.Atoi(pp)
		if err != nil {
			http.Error(w, "Invalid per_page value", http.StatusBadRequest)
			return
		}
		perPage = parsed
	}

	// Return paginated list
	resp := HistoryResponse{
		Conversations: []Conversation{},
		Total:         0,
		Page:          page,
		PerPage:       perPage,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
