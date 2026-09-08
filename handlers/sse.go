package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"mcp-etoolstec-editfiles/auth"
	"mcp-etoolstec-editfiles/logger"
	"mcp-etoolstec-editfiles/mcp"
)

func HandleSSE(w http.ResponseWriter, r *http.Request) {
	if !auth.CheckAuth(r) {
		http.Error(w, "Unauthorized", 401)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sessionID := uuid.New().String()
	ch := make(chan string, 100)
	sess := &mcp.Session{ID: sessionID, Channel: ch}

	mcp.SessionsMu.Lock()
	mcp.Sessions[sessionID] = sess
	mcp.SessionsMu.Unlock()

	logger.Info("Nova sessão SSE criada: %s", sessionID)

	// Send endpoint event
	endpoint := fmt.Sprintf("/mcp?sessionId=%s", sessionID)
	scheme := "https"
	if r.TLS == nil && r.Host == "localhost:8001" {
		scheme = "http"
	}
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
	absURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, endpoint)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", absURL)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", 500)
		return
	}
	flusher.Flush()

	// Keep connection open and stream messages
	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			mcp.SessionsMu.Lock()
			delete(mcp.Sessions, sessionID)
			mcp.SessionsMu.Unlock()
			logger.Info("Sessão SSE fechada: %s", sessionID)
			return
		case <-time.After(15 * time.Second):
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
