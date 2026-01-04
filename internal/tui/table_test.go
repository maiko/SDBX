package tui

import (
	"strings"
	"testing"
)

func TestNewTable(t *testing.T) {
	table := NewTable("Name", "Status", "Health")

	if len(table.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(table.Headers))
	}

	if table.Headers[0] != "Name" {
		t.Errorf("Expected first header 'Name', got %q", table.Headers[0])
	}
}

func TestTableAddRow(t *testing.T) {
	table := NewTable("A", "B")
	table.AddRow("1", "2")
	table.AddRow("3", "4")

	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}
}

func TestTableAddRowPadding(t *testing.T) {
	table := NewTable("A", "B", "C")
	table.AddRow("1") // Should be padded to 3 cells

	if len(table.Rows[0]) != 3 {
		t.Errorf("Expected row to be padded to 3 cells, got %d", len(table.Rows[0]))
	}
}

func TestTableRender(t *testing.T) {
	table := NewTable("Name", "Value")
	table.AddRow("foo", "bar")
	table.AddRow("hello", "world")

	output := table.Render()

	// Check headers present
	if !strings.Contains(output, "Name") {
		t.Error("Output should contain 'Name' header")
	}
	if !strings.Contains(output, "Value") {
		t.Error("Output should contain 'Value' header")
	}

	// Check data present
	if !strings.Contains(output, "foo") {
		t.Error("Output should contain 'foo'")
	}
	if !strings.Contains(output, "world") {
		t.Error("Output should contain 'world'")
	}

	// Check separator present
	if !strings.Contains(output, "─") {
		t.Error("Output should contain separator")
	}
}

func TestTableRenderEmpty(t *testing.T) {
	table := NewTable()
	output := table.Render()

	if output != "" {
		t.Errorf("Empty table should render empty string, got %q", output)
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no escapes here", "no escapes here"},
	}

	for _, tt := range tests {
		result := stripAnsi(tt.input)
		if result != tt.expected {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestStatusBadge(t *testing.T) {
	running := StatusBadge(true)
	if !strings.Contains(running, "Running") {
		t.Error("Running badge should contain 'Running'")
	}

	stopped := StatusBadge(false)
	if !strings.Contains(stopped, "Stopped") {
		t.Error("Stopped badge should contain 'Stopped'")
	}
}

func TestHealthBadge(t *testing.T) {
	healthy := HealthBadge("healthy")
	if !strings.Contains(healthy, "Healthy") {
		t.Error("Healthy badge should contain 'Healthy'")
	}

	unhealthy := HealthBadge("unhealthy")
	if !strings.Contains(unhealthy, "Unhealthy") {
		t.Error("Unhealthy badge should contain 'Unhealthy'")
	}

	starting := HealthBadge("starting")
	if !strings.Contains(starting, "Starting") {
		t.Error("Starting badge should contain 'Starting'")
	}
}

func TestEnabledBadge(t *testing.T) {
	enabled := EnabledBadge(true)
	if !strings.Contains(enabled, "Enabled") {
		t.Error("Enabled badge should contain 'Enabled'")
	}

	disabled := EnabledBadge(false)
	if !strings.Contains(disabled, "Disabled") {
		t.Error("Disabled badge should contain 'Disabled'")
	}
}

func TestServiceTable(t *testing.T) {
	table := ServiceTable()
	if len(table.Headers) != 4 {
		t.Errorf("ServiceTable should have 4 headers, got %d", len(table.Headers))
	}
}

func TestAddonTable(t *testing.T) {
	table := AddonTable()
	if len(table.Headers) != 4 {
		t.Errorf("AddonTable should have 4 headers, got %d", len(table.Headers))
	}
}

func TestSourceTable(t *testing.T) {
	table := SourceTable()
	if len(table.Headers) != 4 {
		t.Errorf("SourceTable should have 4 headers, got %d", len(table.Headers))
	}
}
