package tui

import (
	"strings"
	"testing"
)

func TestRenderProgressBar(t *testing.T) {
	config := DefaultProgressConfig()

	tests := []struct {
		percent  float64
		contains string
	}{
		{0, "0%"},
		{50, "50%"},
		{100, "100%"},
		{-10, "0%"},   // Should clamp to 0
		{150, "100%"}, // Should clamp to 100
	}

	for _, tt := range tests {
		result := RenderProgressBar(tt.percent, config)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("RenderProgressBar(%.0f) should contain %q, got %q", tt.percent, tt.contains, result)
		}
	}
}

func TestRenderProgressBarNoPercent(t *testing.T) {
	config := DefaultProgressConfig()
	config.ShowPercent = false

	result := RenderProgressBar(50, config)
	if strings.Contains(result, "%") {
		t.Error("Progress bar with ShowPercent=false should not contain %")
	}
}

func TestNewStepProgress(t *testing.T) {
	steps := []string{"Step 1", "Step 2", "Step 3"}
	progress := NewStepProgress(steps...)

	if progress.total != 3 {
		t.Errorf("Expected total 3, got %d", progress.total)
	}

	if progress.current != 0 {
		t.Errorf("Expected current 0, got %d", progress.current)
	}
}

func TestStepProgressNext(t *testing.T) {
	progress := NewStepProgress("A", "B", "C")

	progress.Next()
	if progress.current != 1 {
		t.Errorf("After Next(), expected current 1, got %d", progress.current)
	}

	progress.Next()
	if progress.current != 2 {
		t.Errorf("After second Next(), expected current 2, got %d", progress.current)
	}

	// Should not go past the end
	progress.Next()
	if progress.current != 2 {
		t.Errorf("Next() should not advance past last step, got %d", progress.current)
	}
}

func TestStepProgressSetStep(t *testing.T) {
	progress := NewStepProgress("A", "B", "C")

	progress.SetStep(2)
	if progress.current != 2 {
		t.Errorf("Expected current 2, got %d", progress.current)
	}

	// Invalid step should not change
	progress.SetStep(-1)
	if progress.current != 2 {
		t.Errorf("Invalid SetStep should not change current, got %d", progress.current)
	}

	progress.SetStep(100)
	if progress.current != 2 {
		t.Errorf("Invalid SetStep should not change current, got %d", progress.current)
	}
}

func TestStepProgressCurrentStep(t *testing.T) {
	progress := NewStepProgress("First", "Second", "Third")

	if progress.CurrentStep() != "First" {
		t.Errorf("Expected 'First', got %q", progress.CurrentStep())
	}

	progress.Next()
	if progress.CurrentStep() != "Second" {
		t.Errorf("Expected 'Second', got %q", progress.CurrentStep())
	}
}

func TestStepProgressIsComplete(t *testing.T) {
	progress := NewStepProgress("A", "B")

	if progress.IsComplete() {
		t.Error("Progress should not be complete initially")
	}

	progress.Next()
	if !progress.IsComplete() {
		t.Error("Progress should be complete after advancing to last step")
	}
}

func TestStepProgressRender(t *testing.T) {
	progress := NewStepProgress("Domain", "Storage", "Addons")
	output := progress.Render()

	// Check step indicator
	if !strings.Contains(output, "Step 1 of 3") {
		t.Error("Render should contain step indicator")
	}

	// Check all steps are listed
	for _, step := range []string{"Domain", "Storage", "Addons"} {
		if !strings.Contains(output, step) {
			t.Errorf("Render should contain step %q", step)
		}
	}
}

func TestStepProgressRenderCompact(t *testing.T) {
	progress := NewStepProgress("A", "B", "C")
	compact := progress.RenderCompact()

	// Should contain dots/circles
	if !strings.Contains(compact, "●") && !strings.Contains(compact, "○") {
		t.Error("Compact render should contain progress indicators")
	}
}

func TestCheckList(t *testing.T) {
	cl := NewCheckList()

	idx1 := cl.Add("Check 1")
	idx2 := cl.Add("Check 2")

	if idx1 != 0 || idx2 != 1 {
		t.Errorf("Add should return correct indices, got %d and %d", idx1, idx2)
	}

	if len(cl.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(cl.items))
	}
}

func TestCheckListSetStatus(t *testing.T) {
	cl := NewCheckList()
	idx := cl.Add("Test")

	cl.SetStatus(idx, "success", "Passed")

	if cl.items[idx].status != "success" {
		t.Errorf("Expected status 'success', got %q", cl.items[idx].status)
	}

	if cl.items[idx].detail != "Passed" {
		t.Errorf("Expected detail 'Passed', got %q", cl.items[idx].detail)
	}
}

func TestCheckListRender(t *testing.T) {
	cl := NewCheckList()
	cl.Add("Docker")
	cl.SetStatus(0, "success", "Running")

	cl.Add("Network")
	cl.SetStatus(1, "error", "Failed")

	output := cl.Render()

	if !strings.Contains(output, "Docker") {
		t.Error("Render should contain 'Docker'")
	}

	if !strings.Contains(output, "Network") {
		t.Error("Render should contain 'Network'")
	}

	if !strings.Contains(output, "Running") {
		t.Error("Render should contain 'Running' detail")
	}
}
