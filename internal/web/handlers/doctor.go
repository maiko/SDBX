package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/maiko/sdbx/internal/doctor"
)

const doctorRunTimeout = 60 * time.Second

// DoctorHandler handles doctor/diagnostics routes
type DoctorHandler struct {
	projectDir string
	templates  *template.Template
}

// NewDoctorHandler creates a new doctor handler
func NewDoctorHandler(projectDir string, tmpl *template.Template) *DoctorHandler {
	return &DoctorHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// DoctorCheckResult represents a single check result in the JSON response
type DoctorCheckResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Duration string `json:"duration"`
}

// DoctorSummary represents the summary counts in the JSON response
type DoctorSummary struct {
	Passed  int `json:"passed"`
	Warning int `json:"warning"`
	Failed  int `json:"failed"`
}

// DoctorResponse represents the full JSON response for the run-checks endpoint
type DoctorResponse struct {
	Success bool                `json:"success"`
	Checks  []DoctorCheckResult `json:"checks"`
	Summary DoctorSummary       `json:"summary"`
}

// HandleDoctorPage handles the doctor/diagnostics page
func (h *DoctorHandler) HandleDoctorPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), doctorRunTimeout)
	defer cancel()

	doc := doctor.NewDoctor(h.projectDir)
	checks := doc.RunAll(ctx)

	results, summary := buildDoctorResults(checks)

	data := map[string]interface{}{
		"Checks":  results,
		"Summary": summary,
	}

	h.renderTemplate(w, "pages/doctor.html", data)
}

// HandleRunChecks handles POST /api/doctor/run - runs checks and returns JSON
func (h *DoctorHandler) HandleRunChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), doctorRunTimeout)
	defer cancel()

	doc := doctor.NewDoctor(h.projectDir)
	checks := doc.RunAll(ctx)

	results, summary := buildDoctorResults(checks)

	h.respondJSON(w, http.StatusOK, DoctorResponse{
		Success: true,
		Checks:  results,
		Summary: summary,
	})
}

// buildDoctorResults converts doctor.Check results into response types
func buildDoctorResults(checks []doctor.Check) ([]DoctorCheckResult, DoctorSummary) {
	results := make([]DoctorCheckResult, 0, len(checks))
	var summary DoctorSummary

	for _, c := range checks {
		status := checkStatusToString(c.Status)

		switch status {
		case "passed":
			summary.Passed++
		case "warning":
			summary.Warning++
		case "failed":
			summary.Failed++
		}

		results = append(results, DoctorCheckResult{
			Name:     c.Name,
			Status:   status,
			Message:  c.Message,
			Duration: formatDuration(c.Duration),
		})
	}

	return results, summary
}

// checkStatusToString converts a doctor.CheckStatus to its string representation
func checkStatusToString(status doctor.CheckStatus) string {
	switch status {
	case doctor.StatusPassed:
		return "passed"
	case doctor.StatusWarning:
		return "warning"
	case doctor.StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// formatDuration formats a time.Duration for display (e.g., "45ms", "1.2s")
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func (h *DoctorHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *DoctorHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "doctor", data)
}
