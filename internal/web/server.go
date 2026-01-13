package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/web/handlers"
	"github.com/maiko/sdbx/internal/web/middleware"
)

// Server represents the HTTP server
type Server struct {
	config      *ServerConfig
	httpServer  *http.Server
	registry    *registry.Registry
	compose     *docker.Compose
	templates   *template.Template
	setupToken  string
	initialized bool
	dockerMode  bool
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host       string
	Port       int
	ProjectDir string
}

// NewServer creates a new web server instance
func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// Check deployment phase
	if err := s.checkPhase(); err != nil {
		return fmt.Errorf("failed to determine deployment phase: %w", err)
	}

	// Load templates
	if err := s.loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Initialize dependencies
	if err := s.initializeDependencies(); err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.applyMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("\n%s\n", s.formatServerMessage())
		fmt.Printf("Server listening on %s\n", addr)
		fmt.Println("Press Ctrl+C to stop")
		fmt.Println()

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return s.Shutdown(30 * time.Second)
	case err := <-errCh:
		return err
	}
}

// checkPhase determines the deployment phase and generates setup token if needed
func (s *Server) checkPhase() error {
	// Check for .sdbx.yaml existence
	configPath := filepath.Join(s.config.ProjectDir, ".sdbx.yaml")
	_, err := os.Stat(configPath)
	s.initialized = err == nil

	// Check if running in Docker
	s.dockerMode = os.Getenv("SDBX_MODE") == "server"

	// Generate one-time setup token if pre-init
	if !s.initialized {
		token, err := generateSecureToken(32) // 32 bytes = 256 bits
		if err != nil {
			return fmt.Errorf("failed to generate setup token: %w", err)
		}
		s.setupToken = token
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// getLocalIP attempts to determine the server's local IP address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

// formatServerMessage formats the server startup message
func (s *Server) formatServerMessage() string {
	msg := "SDBX Web UI Server Starting...\n\n"

	if !s.initialized {
		// Pre-init mode: display setup token URL
		host := getLocalIP()
		if s.config.Host == "0.0.0.0" {
			msg += "Setup wizard available at:\n"
			msg += fmt.Sprintf("  http://%s:%d?token=%s\n\n", host, s.config.Port, s.setupToken)
			msg += "Token expires after setup completion or server restart.\n"
			msg += "Access this URL from any device on your network.\n\n"
		} else {
			msg += "Setup wizard available at:\n"
			msg += fmt.Sprintf("  http://%s:%d?token=%s\n\n", s.config.Host, s.config.Port, s.setupToken)
		}
	} else {
		// Post-init mode
		if s.dockerMode {
			msg += "Running in production mode (Docker service)\n"
			msg += "Access via configured domain through Authelia\n\n"
		} else {
			msg += "Running in development mode\n"
			msg += "⚠ Warning: For development only, use Docker service in production\n\n"
		}
	}

	return msg
}

// loadTemplates loads and parses all HTML templates
func (s *Server) loadTemplates() error {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Walk the embedded filesystem to find all .html files
	// Note: ParseFS with "**" glob doesn't work in Go - must walk manually
	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Only process .html files
		if strings.HasSuffix(path, ".html") {
			content, err := fs.ReadFile(templatesFS, path)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", path, err)
			}
			// Strip "templates/" prefix so handlers can use shorter paths
			// e.g., "templates/pages/setup/welcome.html" -> "pages/setup/welcome.html"
			templateName := strings.TrimPrefix(path, "templates/")
			_, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	s.templates = tmpl
	return nil
}

// initializeDependencies initializes server dependencies
func (s *Server) initializeDependencies() error {
	// Initialize registry
	reg, err := registry.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}
	s.registry = reg

	// Initialize Docker Compose (only if initialized)
	if s.initialized {
		s.compose = docker.NewCompose(s.config.ProjectDir)
	}

	return nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Serve static files
	staticFS, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	if !s.initialized {
		// Pre-init routes: Setup wizard
		setupHandler := handlers.NewSetupHandler(s.registry, s.config.ProjectDir, s.templates)
		mux.HandleFunc("/", setupHandler.HandleWelcome)
		mux.HandleFunc("/setup/domain", setupHandler.HandleDomain)
		mux.HandleFunc("/setup/cloudflare", setupHandler.HandleCloudflareTokenForm)
		mux.HandleFunc("/setup/admin", setupHandler.HandleAdmin)
		mux.HandleFunc("/setup/storage", setupHandler.HandleStorage)
		mux.HandleFunc("/setup/vpn", setupHandler.HandleVPN)
		mux.HandleFunc("/setup/addons", setupHandler.HandleAddons)
		mux.HandleFunc("/setup/summary", setupHandler.HandleSummary)
		mux.HandleFunc("/setup/complete", setupHandler.HandleComplete)
	} else {
		// Post-init routes: Dashboard and management
		dashboardHandler := handlers.NewDashboardHandler(s.compose, s.registry, s.templates)
		servicesHandler := handlers.NewServicesHandler(s.compose, s.registry, s.templates)
		logsHandler := handlers.NewLogsHandler(s.compose, s.registry, s.templates)
		addonsHandler := handlers.NewAddonsHandler(s.registry, s.config.ProjectDir, s.templates)
		configHandler := handlers.NewConfigHandler(s.config.ProjectDir, s.templates)
		backupHandler := handlers.NewBackupHandler(s.config.ProjectDir, s.templates)
		serviceInfoHandler := handlers.NewServiceInfoHandler(s.registry, s.templates)

		// Pages
		mux.HandleFunc("/", dashboardHandler.HandleDashboard)
		mux.HandleFunc("/services", servicesHandler.HandleServicesPage)
		mux.HandleFunc("/service-info", serviceInfoHandler.HandleServiceInfoPage)
		mux.HandleFunc("/logs/{service}", logsHandler.HandleLogsPage)
		mux.HandleFunc("/addons", addonsHandler.HandleAddonsPage)
		mux.HandleFunc("/config", configHandler.HandleConfigPage)
		mux.HandleFunc("/backup", backupHandler.HandleBackupPage)

		// API endpoints
		mux.HandleFunc("/api/services", servicesHandler.HandleGetServices)
		mux.HandleFunc("/api/services/{service}/start", servicesHandler.HandleStartService)
		mux.HandleFunc("/api/services/{service}/stop", servicesHandler.HandleStopService)
		mux.HandleFunc("/api/services/{service}/restart", servicesHandler.HandleRestartService)

		// Log endpoints
		mux.HandleFunc("/api/logs/{service}", logsHandler.HandleGetLogs)
		mux.HandleFunc("/api/logs/{service}/stream", logsHandler.HandleLogStream)

		// Addon endpoints
		mux.HandleFunc("/api/addons/search", addonsHandler.HandleSearchAddons)
		mux.HandleFunc("/api/addons/{addon}/enable", addonsHandler.HandleEnableAddon)
		mux.HandleFunc("/api/addons/{addon}/disable", addonsHandler.HandleDisableAddon)

		// Config endpoints
		mux.HandleFunc("/api/config", configHandler.HandleGetConfig)
		mux.HandleFunc("/api/config/validate", configHandler.HandleValidateConfig)
		mux.HandleFunc("/api/config/save", configHandler.HandleSaveConfig)

		// Backup endpoints
		mux.HandleFunc("/api/backup/list", backupHandler.HandleListBackups)
		mux.HandleFunc("/api/backup/create", backupHandler.HandleCreateBackup)
		mux.HandleFunc("/api/backup/restore/{name}", backupHandler.HandleRestoreBackup)
		mux.HandleFunc("/api/backup/delete/{name}", backupHandler.HandleDeleteBackup)
	}
}

// applyMiddleware applies middleware to the handler chain
func (s *Server) applyMiddleware(handler http.Handler) http.Handler {
	// Recovery middleware (outermost)
	handler = middleware.Recovery(handler)

	// Logging middleware
	handler = middleware.Logging(handler)

	// Auth middleware (based on phase)
	authMiddleware := middleware.NewAuth(s.initialized, s.dockerMode, s.setupToken)
	handler = authMiddleware.Middleware(handler)

	return handler
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(timeout time.Duration) error {
	if s.httpServer == nil {
		return nil
	}

	fmt.Println("\nShutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("Server stopped gracefully")
	return nil
}

// Run runs the server with signal handling
func Run(cfg *ServerConfig) error {
	server := NewServer(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		cancel()
		return server.Shutdown(30 * time.Second)
	case err := <-errCh:
		return err
	}
}
