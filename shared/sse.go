package shared

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var (
	sseClients = map[chan string]bool{}
	sseMu      sync.Mutex
)

// AutoInputHandler handles broadcasting words to all connected SSE clients
func AutoInputHandler(w http.ResponseWriter, r *http.Request) {
	word := strings.ToLower(r.URL.Query().Get("q"))
	if word == "" {
		return
	}

	// Broadcast to all SSE clients
	sseMu.Lock()
	for ch := range sseClients {
		select {
		case ch <- word:
		default:
		}
	}
	sseMu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// SSEHandler handles the EventSource connections from the Roblox client
func SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 5)
	sseMu.Lock()
	sseClients[ch] = true
	sseMu.Unlock()

	defer func() {
		sseMu.Lock()
		delete(sseClients, ch)
		sseMu.Unlock()
	}()

	for {
		select {
		case word := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", word)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// GetConnectedClientsCount returns the number of active SSE connections
func GetConnectedClientsCount() int {
	sseMu.Lock()
	defer sseMu.Unlock()
	return len(sseClients)
}
