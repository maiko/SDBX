package handlers

import (
	"testing"
	"time"

	"github.com/maiko/sdbx/internal/doctor"
)

func TestCheckStatusToString(t *testing.T) {
	tests := []struct {
		input    doctor.CheckStatus
		expected string
	}{
		{doctor.StatusPassed, "passed"},
		{doctor.StatusWarning, "warning"},
		{doctor.StatusFailed, "failed"},
		{doctor.StatusPending, "unknown"},
		{doctor.StatusRunning, "unknown"},
		{doctor.CheckStatus(99), "unknown"},
	}

	for _, tt := range tests {
		result := checkStatusToString(tt.input)
		if result != tt.expected {
			t.Errorf("checkStatusToString(%d) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{450 * time.Microsecond, "450us"},
		{0, "0us"},
		{45 * time.Millisecond, "45ms"},
		{999 * time.Millisecond, "999ms"},
		{1500 * time.Millisecond, "1.5s"},
		{10 * time.Second, "10.0s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestBuildDoctorResults(t *testing.T) {
	checks := []doctor.Check{
		{Name: "Docker", Status: doctor.StatusPassed, Message: "OK", Duration: 10 * time.Millisecond},
		{Name: "Disk", Status: doctor.StatusWarning, Message: "Low", Duration: 5 * time.Millisecond},
		{Name: "VPN", Status: doctor.StatusFailed, Message: "Down", Duration: 1 * time.Second},
	}

	results, summary := buildDoctorResults(checks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Status != "passed" {
		t.Errorf("expected first check status 'passed', got %q", results[0].Status)
	}
	if results[1].Status != "warning" {
		t.Errorf("expected second check status 'warning', got %q", results[1].Status)
	}
	if results[2].Status != "failed" {
		t.Errorf("expected third check status 'failed', got %q", results[2].Status)
	}

	if summary.Passed != 1 || summary.Warning != 1 || summary.Failed != 1 {
		t.Errorf("summary = %+v, expected 1/1/1", summary)
	}
}

func TestBuildDoctorResultsEmpty(t *testing.T) {
	results, summary := buildDoctorResults(nil)

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
	if summary.Passed != 0 || summary.Warning != 0 || summary.Failed != 0 {
		t.Errorf("expected all zeros, got %+v", summary)
	}
}

func TestNewDoctorHandler(t *testing.T) {
	h := NewDoctorHandler("/tmp/test", nil)
	if h.projectDir != "/tmp/test" {
		t.Errorf("expected projectDir '/tmp/test', got %q", h.projectDir)
	}
}
