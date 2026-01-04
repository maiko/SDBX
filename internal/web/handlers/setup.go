package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/argon2"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/generator"
	"github.com/maiko/sdbx/internal/registry"
)

// SetupHandler handles the setup wizard
type SetupHandler struct {
	registry   *registry.Registry
	projectDir string
	templates  *template.Template
	sessions   map[string]*WizardSession
	mu         sync.RWMutex
}

// WizardSession holds the state of a setup wizard session
type WizardSession struct {
	Config   *config.Config
	Password string // Temporary storage for password (cleared after hashing)
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(reg *registry.Registry, projectDir string, tmpl *template.Template) *SetupHandler {
	return &SetupHandler{
		registry:   reg,
		projectDir: projectDir,
		templates:  tmpl,
		sessions:   make(map[string]*WizardSession),
	}
}

// getSession retrieves or creates a session for the given session ID (from cookie)
func (h *SetupHandler) getSession(r *http.Request) (*WizardSession, string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Try to get session ID from cookie
	cookie, err := r.Cookie("wizard_session")
	var sessionID string
	if err == nil {
		sessionID = cookie.Value
		if session, exists := h.sessions[sessionID]; exists {
			return session, sessionID
		}
	}

	// Create new session
	sessionID = generateSessionID()
	session := &WizardSession{
		Config: config.DefaultConfig(),
	}
	h.sessions[sessionID] = session

	return session, sessionID
}

// deleteSession removes a session
func (h *SetupHandler) deleteSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.sessions, sessionID)
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// setSessionCookie sets the session cookie
func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "wizard_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   1800, // 30 minutes
	})
}

// HandleWelcome handles the welcome page (step 0)
func (h *SetupHandler) HandleWelcome(w http.ResponseWriter, r *http.Request) {
	_, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodGet {
		// Check if existing project
		hasExisting := false
		entries, err := os.ReadDir(h.projectDir)
		if err == nil {
			for _, entry := range entries {
				if entry.Name() == "compose.yaml" || entry.Name() == ".sdbx.yaml" {
					hasExisting = true
					break
				}
			}
		}

		data := map[string]interface{}{
			"HasExisting": hasExisting,
		}
		h.renderTemplate(w, "pages/setup/welcome.html", data)
	}
}

// HandleDomain handles domain configuration (step 1)
func (h *SetupHandler) HandleDomain(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		domain := r.FormValue("domain")
		exposeMode := r.FormValue("expose_mode")
		routingStrategy := r.FormValue("routing_strategy")
		baseDomain := r.FormValue("base_domain")

		// Validate
		if domain == "" {
			http.Error(w, "Domain is required", http.StatusBadRequest)
			return
		}

		// Update session config
		session.Config.Domain = domain
		session.Config.Expose.Mode = exposeMode
		session.Config.Routing.Strategy = routingStrategy
		if routingStrategy == config.RoutingStrategyPath {
			session.Config.Routing.BaseDomain = baseDomain
		}

		// Redirect to next step
		w.Header().Set("HX-Redirect", "/setup/admin")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Show form
	data := map[string]interface{}{
		"Config": session.Config,
	}
	h.renderTemplate(w, "pages/setup/domain.html", data)
}

// HandleAdmin handles admin credentials (step 2)
func (h *SetupHandler) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate
		if username == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}
		if len(password) < 4 {
			http.Error(w, "Password must be at least 4 characters", http.StatusBadRequest)
			return
		}
		if password != confirmPassword {
			http.Error(w, "Passwords do not match", http.StatusBadRequest)
			return
		}

		// Update session
		session.Config.AdminUser = username

		// Hash password
		hash, err := generateArgon2Hash(password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		session.Config.AdminPasswordHash = hash

		// Redirect to next step
		w.Header().Set("HX-Redirect", "/setup/storage")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Show form
	data := map[string]interface{}{
		"Config": session.Config,
	}
	h.renderTemplate(w, "pages/setup/admin.html", data)
}

// HandleStorage handles storage paths configuration (step 3)
func (h *SetupHandler) HandleStorage(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		mediaPath := r.FormValue("media_path")
		downloadsPath := r.FormValue("downloads_path")
		configPath := r.FormValue("config_path")
		timezone := r.FormValue("timezone")

		// Validate
		if mediaPath == "" || downloadsPath == "" || configPath == "" {
			http.Error(w, "All paths are required", http.StatusBadRequest)
			return
		}

		// Update session
		session.Config.MediaPath = mediaPath
		session.Config.DownloadsPath = downloadsPath
		session.Config.ConfigPath = configPath
		session.Config.Timezone = timezone

		// Redirect to next step
		w.Header().Set("HX-Redirect", "/setup/vpn")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Show form
	data := map[string]interface{}{
		"Config": session.Config,
	}
	h.renderTemplate(w, "pages/setup/storage.html", data)
}

// HandleVPN handles VPN configuration (step 4)
func (h *SetupHandler) HandleVPN(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		vpnEnabled := r.FormValue("vpn_enabled") == "true"
		vpnProvider := r.FormValue("vpn_provider")
		vpnCountry := r.FormValue("vpn_country")

		// Update session
		session.Config.VPNEnabled = vpnEnabled
		if vpnEnabled {
			session.Config.VPNProvider = vpnProvider
			session.Config.VPNCountry = vpnCountry
		}

		// Redirect to next step
		w.Header().Set("HX-Redirect", "/setup/addons")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Show form
	data := map[string]interface{}{
		"Config": session.Config,
		"VPNProviders": []string{
			"nordvpn", "mullvad", "pia", "surfshark", "protonvpn",
			"expressvpn", "windscribe", "ipvanish", "cyberghost", "ivpn",
			"torguard", "vyprvpn", "purevpn", "hidemyass", "perfectprivacy",
			"airvpn", "custom",
		},
	}
	h.renderTemplate(w, "pages/setup/vpn.html", data)
}

// HandleAddons handles addon selection (step 5)
func (h *SetupHandler) HandleAddons(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get selected addons (checkboxes)
		selectedAddons := r.Form["addons"]
		session.Config.Addons = selectedAddons

		// Redirect to summary
		w.Header().Set("HX-Redirect", "/setup/summary")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Load addons from registry
	ctx := r.Context()
	services, err := h.registry.ListServices(ctx)
	if err != nil {
		http.Error(w, "Failed to load addons", http.StatusInternalServerError)
		return
	}

	// Filter addons only
	var addons []AddonInfo
	for _, svc := range services {
		if svc.IsAddon {
			addons = append(addons, AddonInfo{
				Name:        svc.Name,
				Description: svc.Description,
				Category:    string(svc.Category),
			})
		}
	}

	data := map[string]interface{}{
		"Config": session.Config,
		"Addons": addons,
	}
	h.renderTemplate(w, "pages/setup/addons.html", data)
}

// AddonInfo holds addon information for rendering
type AddonInfo struct {
	Name        string
	Description string
	Category    string
}

// HandleSummary handles the configuration summary (step 6)
func (h *SetupHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method == http.MethodPost {
		// User confirmed, redirect to final generation
		w.Header().Set("HX-Redirect", "/setup/complete")
		w.WriteHeader(http.StatusOK)
		return
	}

	// GET: Show summary
	data := map[string]interface{}{
		"Config": session.Config,
	}
	h.renderTemplate(w, "pages/setup/summary.html", data)
}

// HandleComplete handles project generation (final step)
func (h *SetupHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	session, sessionID := h.getSession(r)
	setSessionCookie(w, sessionID)

	if r.Method != http.MethodPost {
		// Show completion UI with "Generate" button
		data := map[string]interface{}{
			"Config": session.Config,
		}
		h.renderTemplate(w, "pages/setup/complete.html", data)
		return
	}

	// POST: Generate project
	gen := generator.NewGeneratorWithRegistry(session.Config, h.projectDir, h.registry)
	if err := gen.Generate(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate project: %v", err), http.StatusInternalServerError)
		return
	}

	// Create data directories if paths are relative
	if !filepath.IsAbs(session.Config.MediaPath) {
		if err := gen.CreateDataDirs(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create directories: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Clear session
	h.deleteSession(sessionID)

	// Return success HTML fragment (htmx will swap into #generation-status)
	w.Header().Set("Content-Type", "text/html")
	if err := h.templates.ExecuteTemplate(w, "pages/setup/complete-success.html", nil); err != nil {
		http.Error(w, "Failed to render success template", http.StatusInternalServerError)
	}
}

// renderTemplate renders a page template wrapped in the wizard layout
func (h *SetupHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	// Clone the template set to avoid modifying the original
	tmpl, err := h.templates.Clone()
	if err != nil {
		httpError(w, "template clone", err, http.StatusInternalServerError)
		return
	}

	// Create a wrapper template that includes the page content in the wizard layout
	wrapperTmpl := `{{define "page-content"}}{{template "` + name + `" .}}{{end}}{{template "layouts/wizard.html" .}}`

	_, err = tmpl.Parse(wrapperTmpl)
	if err != nil {
		httpError(w, "template parse", err, http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		httpError(w, "setup template render", err, http.StatusInternalServerError)
	}
}

// generateArgon2Hash generates an Argon2id hash compatible with Authelia
func generateArgon2Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Authelia defaults: time=3, memory=64MB, threads=4, keyLen=32
	time := uint32(3)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, b64Salt, b64Hash), nil
}
