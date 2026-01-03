package handlers

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
)

// LogsHandler handles log viewing routes
type LogsHandler struct {
	compose   *docker.Compose
	registry  *registry.Registry
	templates *template.Template
	upgrader  websocket.Upgrader
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(compose *docker.Compose, reg *registry.Registry, tmpl *template.Template) *LogsHandler {
	return &LogsHandler{
		compose:  compose,
		registry: reg,
		templates: tmpl,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (same-origin in production)
				return true
			},
		},
	}
}

// LogMessage represents a log message sent via WebSocket
type LogMessage struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Line      string `json:"line"`
}

// HandleLogsPage handles the logs viewer page
func (h *LogsHandler) HandleLogsPage(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}

	// Get service info from registry
	ctx := context.Background()
	services, err := h.registry.ListServices(ctx)
	if err != nil {
		http.Error(w, "Failed to load services", http.StatusInternalServerError)
		return
	}

	var serviceInfo *registry.ServiceInfo
	for _, svc := range services {
		if svc.Name == serviceName {
			serviceInfo = &svc
			break
		}
	}

	if serviceInfo == nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Service":     serviceInfo,
		"ServiceName": serviceName,
	}

	h.renderTemplate(w, "pages/logs.html", data)
}

// HandleLogStream handles WebSocket log streaming
func (h *LogsHandler) HandleLogStream(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("WebSocket upgrade failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start streaming logs
	cmd, err := h.compose.LogsStream(ctx, serviceName, 100)
	if err != nil {
		conn.WriteJSON(map[string]string{
			"error": fmt.Sprintf("Failed to start log stream: %v", err),
		})
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteJSON(map[string]string{
			"error": fmt.Sprintf("Failed to get stdout pipe: %v", err),
		})
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		conn.WriteJSON(map[string]string{
			"error": fmt.Sprintf("Failed to get stderr pipe: %v", err),
		})
		return
	}

	if err := cmd.Start(); err != nil {
		conn.WriteJSON(map[string]string{
			"error": fmt.Sprintf("Failed to start command: %v", err),
		})
		return
	}

	// Handle client disconnection
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel() // Cancel context to stop log streaming
				return
			}
		}
	}()

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			msg := LogMessage{
				Timestamp: time.Now().Format("15:04:05"),
				Service:   serviceName,
				Line:      scanner.Text(),
			}
			if err := conn.WriteJSON(msg); err != nil {
				cancel()
				return
			}
		}
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			msg := LogMessage{
				Timestamp: time.Now().Format("15:04:05"),
				Service:   serviceName,
				Line:      scanner.Text(),
			}
			if err := conn.WriteJSON(msg); err != nil {
				cancel()
				return
			}
		}
	}()

	// Wait for command to finish
	cmd.Wait()
}

// HandleGetLogs handles HTTP GET for recent logs
func (h *LogsHandler) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		http.Error(w, "Service name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get last 100 lines
	logs, err := h.compose.Logs(ctx, serviceName, 100, false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(logs))
}

// renderTemplate renders a template with data
func (h *LogsHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
