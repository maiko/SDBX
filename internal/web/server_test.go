package web

import (
	"html/template"
	"io/fs"
	"strings"
	"testing"
)

// TestTemplateLoading verifies that all templates are loaded correctly
func TestTemplateLoading(t *testing.T) {
	// Load templates using the same method as server
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tmpl, err := loadAllTemplates(funcMap)
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	// List of templates that must exist (without templates/ prefix)
	requiredTemplates := []string{
		// Layouts
		"layouts/base.html",
		"layouts/wizard.html",
		// Setup pages
		"pages/setup/welcome.html",
		"pages/setup/domain.html",
		"pages/setup/admin.html",
		"pages/setup/storage.html",
		"pages/setup/vpn.html",
		"pages/setup/addons.html",
		"pages/setup/summary.html",
		"pages/setup/complete.html",
		// Dashboard pages
		"pages/dashboard.html",
		"pages/services.html",
		"pages/logs.html",
		"pages/addons.html",
		"pages/config.html",
		"pages/integration.html",
		"pages/backup.html",
	}

	for _, name := range requiredTemplates {
		tmplFound := tmpl.Lookup(name)
		if tmplFound == nil {
			t.Errorf("template %q not found", name)
		}
	}
}

// TestEmbeddedFS verifies that the embedded filesystem contains expected files
func TestEmbeddedFS(t *testing.T) {
	// Check that templatesFS is accessible
	entries, err := fs.ReadDir(templatesFS, "templates")
	if err != nil {
		t.Fatalf("failed to read templates directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("templates directory is empty")
	}

	// Check for key subdirectories
	expectedDirs := []string{"layouts", "pages", "components"}
	for _, dir := range expectedDirs {
		_, err := fs.ReadDir(templatesFS, "templates/"+dir)
		if err != nil {
			t.Errorf("expected directory templates/%s not found: %v", dir, err)
		}
	}
}

// loadAllTemplates loads all templates from the embedded FS
// This mirrors the server's loadTemplates method
func loadAllTemplates(funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("").Funcs(funcMap)

	// Walk the embedded filesystem to find all .html files
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
				return err
			}
			// Strip "templates/" prefix to match handler expectations
			templateName := strings.TrimPrefix(path, "templates/")
			_, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return err
			}
		}
		return nil
	})

	return tmpl, err
}
