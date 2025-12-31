package tui

import (
	"strings"
	"testing"
)

func TestColorPalette(t *testing.T) {
	// Verify color constants are set
	colors := map[string]string{
		"ColorPrimary": string(ColorPrimary),
		"ColorSuccess": string(ColorSuccess),
		"ColorWarning": string(ColorWarning),
		"ColorError":   string(ColorError),
		"ColorInfo":    string(ColorInfo),
		"ColorMuted":   string(ColorMuted),
		"ColorWhite":   string(ColorWhite),
		"ColorBlack":   string(ColorBlack),
	}

	for name, color := range colors {
		if color == "" {
			t.Errorf("%s should not be empty", name)
		}
		if !strings.HasPrefix(color, "#") {
			t.Errorf("%s = %s, should start with #", name, color)
		}
	}
}

func TestStatusIcons(t *testing.T) {
	icons := map[string]string{
		"IconSuccess": IconSuccess,
		"IconError":   IconError,
		"IconWarning": IconWarning,
		"IconInfo":    IconInfo,
		"IconRunning": IconRunning,
		"IconStopped": IconStopped,
		"IconSpinner": IconSpinner,
		"IconArrow":   IconArrow,
		"IconCheck":   IconCheck,
		"IconCross":   IconCross,
		"IconLock":    IconLock,
		"IconUnlock":  IconUnlock,
		"IconStar":    IconStar,
		"IconDot":     IconDot,
		"IconDash":    IconDash,
		"IconBox":     IconBox,
		"IconFolder":  IconFolder,
		"IconGear":    IconGear,
		"IconNetwork": IconNetwork,
		"IconDocker":  IconDocker,
		"IconKey":     IconKey,
		"IconRocket":  IconRocket,
		"IconPackage": IconPackage,
	}

	for name, icon := range icons {
		if icon == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestLogo(t *testing.T) {
	if Logo == "" {
		t.Error("Logo should not be empty")
	}
	if !strings.Contains(Logo, "███") {
		t.Error("Logo should contain ASCII art blocks")
	}
}

func TestLogoStyled(t *testing.T) {
	styled := LogoStyled()
	if styled == "" {
		t.Error("LogoStyled() should not return empty string")
	}
	// The styled version should contain ANSI escape codes for color
	if !strings.Contains(styled, "\x1b[") && !strings.Contains(styled, "███") {
		t.Error("LogoStyled() should contain either ANSI codes or the logo content")
	}
}

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		name     string
		healthy  bool
		expected string
	}{
		{"healthy service", true, "Healthy"},
		{"unhealthy service", false, "Unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderStatus(tt.healthy)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("RenderStatus(%v) should contain '%s', got: %s", tt.healthy, tt.expected, result)
			}
			// Should contain an icon
			if !strings.Contains(result, IconSuccess) && !strings.Contains(result, IconError) {
				t.Error("RenderStatus() should contain a status icon")
			}
		})
	}
}

func TestRenderServiceStatus(t *testing.T) {
	tests := []struct {
		name          string
		serviceName   string
		running       bool
		healthy       bool
		expectedIcon  string
		shouldContain string
	}{
		{
			name:          "stopped service",
			serviceName:   "traefik",
			running:       false,
			healthy:       false,
			expectedIcon:  IconStopped,
			shouldContain: "traefik",
		},
		{
			name:          "running and healthy",
			serviceName:   "authelia",
			running:       true,
			healthy:       true,
			expectedIcon:  IconRunning,
			shouldContain: "authelia",
		},
		{
			name:          "running but unhealthy",
			serviceName:   "plex",
			running:       true,
			healthy:       false,
			expectedIcon:  IconRunning,
			shouldContain: "plex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderServiceStatus(tt.serviceName, tt.running, tt.healthy)

			if !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Result should contain service name '%s': %s", tt.shouldContain, result)
			}

			// Check for presence of icon (ANSI codes may be present)
			if !strings.Contains(result, tt.expectedIcon) {
				t.Errorf("Result should contain icon '%s': %s", tt.expectedIcon, result)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		width   int
		filled  int
		empty   int
	}{
		{"empty bar", 0.0, 10, 0, 10},
		{"quarter full", 0.25, 20, 5, 15},
		{"half full", 0.5, 10, 5, 5},
		{"three quarters", 0.75, 20, 15, 5},
		{"full bar", 1.0, 10, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProgressBar(tt.percent, tt.width)

			// Count filled and empty characters (ignoring ANSI codes)
			// Remove ANSI escape sequences for counting
			clean := strings.ReplaceAll(result, "\x1b[", "")
			clean = strings.ReplaceAll(clean, "m", "")

			filledCount := strings.Count(clean, "█")
			emptyCount := strings.Count(clean, "░")

			// Allow for ANSI codes - the actual bar content should match
			if filledCount < tt.filled-1 || filledCount > tt.filled+1 {
				t.Errorf("Expected ~%d filled chars, got %d", tt.filled, filledCount)
			}
			if emptyCount < tt.empty-1 || emptyCount > tt.empty+1 {
				t.Errorf("Expected ~%d empty chars, got %d", tt.empty, emptyCount)
			}
		})
	}
}

func TestProgressBarEdgeCases(t *testing.T) {
	// Test with zero width
	result := ProgressBar(0.5, 0)
	if result != "" {
		t.Errorf("Expected empty result for zero width, got: %s", result)
	}

	// Test with negative percent (should handle gracefully)
	result = ProgressBar(-0.1, 10)
	// Should not panic

	// Test with > 100% (should handle gracefully)
	result = ProgressBar(1.5, 10)
	// Should not panic
}

func TestRenderSuccessBox(t *testing.T) {
	title := "Operation Complete"
	message := "All services are running"

	result := RenderSuccessBox(title, message)

	// Should contain the title
	if !strings.Contains(result, title) {
		t.Errorf("Result should contain title '%s'", title)
	}

	// Should contain the message
	if !strings.Contains(result, message) {
		t.Errorf("Result should contain message '%s'", message)
	}

	// Should contain success icon
	if !strings.Contains(result, IconSuccess) {
		t.Error("Result should contain success icon")
	}

	// Should not be empty
	if result == "" {
		t.Error("RenderSuccessBox should not return empty string")
	}
}

func TestRenderSuccessBoxWithEmptyStrings(t *testing.T) {
	// Test with empty title
	result := RenderSuccessBox("", "message")
	if !strings.Contains(result, "message") {
		t.Error("Should still render message")
	}

	// Test with empty message
	result = RenderSuccessBox("title", "")
	if !strings.Contains(result, "title") {
		t.Error("Should still render title")
	}

	// Test with both empty
	result = RenderSuccessBox("", "")
	// Should still render the box structure
	if result == "" {
		t.Error("Should render empty box")
	}
}

func TestStylesAreInitialized(t *testing.T) {
	// Verify that all style variables are initialized (not nil/zero)
	styles := []struct {
		name  string
		style interface{}
	}{
		{"TitleStyle", TitleStyle},
		{"SubtitleStyle", SubtitleStyle},
		{"BoxStyle", BoxStyle},
		{"SuccessStyle", SuccessStyle},
		{"WarningStyle", WarningStyle},
		{"ErrorStyle", ErrorStyle},
		{"InfoStyle", InfoStyle},
		{"MutedStyle", MutedStyle},
		{"SelectedStyle", SelectedStyle},
		{"FocusedStyle", FocusedStyle},
		{"TableHeaderStyle", TableHeaderStyle},
		{"TableCellStyle", TableCellStyle},
	}

	for _, s := range styles {
		if s.style == nil {
			t.Errorf("%s should be initialized", s.name)
		}
	}
}

func TestRenderFunctions(t *testing.T) {
	// Test that rendering functions don't panic
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "RenderStatus with true",
			fn:   func() { _ = RenderStatus(true) },
		},
		{
			name: "RenderStatus with false",
			fn:   func() { _ = RenderStatus(false) },
		},
		{
			name: "RenderServiceStatus",
			fn:   func() { _ = RenderServiceStatus("test", true, true) },
		},
		{
			name: "ProgressBar",
			fn:   func() { _ = ProgressBar(0.5, 10) },
		},
		{
			name: "RenderSuccessBox",
			fn:   func() { _ = RenderSuccessBox("Test", "Message") },
		},
		{
			name: "LogoStyled",
			fn:   func() { _ = LogoStyled() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Function %s panicked: %v", tt.name, r)
				}
			}()
			tt.fn()
		})
	}
}

func TestRenderInfoBox(t *testing.T) {
	result := RenderInfoBox("Information", "This is an info message")
	if !strings.Contains(result, "Information") {
		t.Error("Result should contain title")
	}
	if !strings.Contains(result, "info message") {
		t.Error("Result should contain message")
	}
}

func TestRenderErrorBox(t *testing.T) {
	result := RenderErrorBox("Error", "Something went wrong")
	if !strings.Contains(result, "Error") {
		t.Error("Result should contain title")
	}
	if !strings.Contains(result, "went wrong") {
		t.Error("Result should contain message")
	}
}

func TestRenderWarningBox(t *testing.T) {
	result := RenderWarningBox("Warning", "This is a warning")
	if !strings.Contains(result, "Warning") {
		t.Error("Result should contain title")
	}
	if !strings.Contains(result, "warning") {
		t.Error("Result should contain message")
	}
}

func TestRenderSection(t *testing.T) {
	result := RenderSection("Configuration")
	if !strings.Contains(result, "Configuration") {
		t.Error("Result should contain section name")
	}
}

func TestRenderKeyValue(t *testing.T) {
	result := RenderKeyValue("Version", "1.0.0")
	if !strings.Contains(result, "Version") {
		t.Error("Result should contain key")
	}
	if !strings.Contains(result, "1.0.0") {
		t.Error("Result should contain value")
	}
}

func TestRenderBullet(t *testing.T) {
	result := RenderBullet("List item")
	if !strings.Contains(result, "List item") {
		t.Error("Result should contain text")
	}
	if !strings.Contains(result, IconDot) {
		t.Error("Result should contain bullet icon")
	}
}

func TestRenderCommand(t *testing.T) {
	result := RenderCommand("sdbx up")
	if !strings.Contains(result, "sdbx up") {
		t.Error("Result should contain command")
	}
}

func TestRenderDivider(t *testing.T) {
	result := RenderDivider(10)
	if len(result) == 0 {
		t.Error("Result should not be empty")
	}
}

func TestRenderHeader(t *testing.T) {
	// Without subtitle
	result := RenderHeader("Title", "")
	if !strings.Contains(result, "Title") {
		t.Error("Result should contain title")
	}

	// With subtitle
	result = RenderHeader("Title", "Subtitle")
	if !strings.Contains(result, "Title") {
		t.Error("Result should contain title")
	}
	if !strings.Contains(result, "Subtitle") {
		t.Error("Result should contain subtitle")
	}
}

func TestRenderCategory(t *testing.T) {
	categories := []string{"media", "downloads", "management", "utility", "networking", "auth", "unknown"}

	for _, cat := range categories {
		result := RenderCategory(cat)
		if !strings.Contains(result, cat) {
			t.Errorf("Result should contain category name '%s'", cat)
		}
	}
}

func TestCategoryColors(t *testing.T) {
	expectedCategories := []string{"media", "downloads", "management", "utility", "networking", "auth"}

	for _, cat := range expectedCategories {
		if _, ok := CategoryColors[cat]; !ok {
			t.Errorf("CategoryColors should have color for '%s'", cat)
		}
	}
}

func TestSpinnerFrames(t *testing.T) {
	if len(SpinnerFrames) == 0 {
		t.Error("SpinnerFrames should not be empty")
	}

	for i, frame := range SpinnerFrames {
		if frame == "" {
			t.Errorf("SpinnerFrame[%d] should not be empty", i)
		}
	}
}
